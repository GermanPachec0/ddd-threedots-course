package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/scoreboard"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type PlayerJoinRequest struct {
	Name string `json:"name"`
	Team string `json:"team"`
}

type PlayerJoined struct {
	ID string `json:"id"`
}

type TeamCreated struct {
	ID string `json:"id"`
}

type Player struct {
	ID   string
	Name string
	Team string
}

type Team struct {
	ID      string
	Name    string
	Players []Player
}

func main() {
	logger := watermill.NewStdLogger(false, false)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

	rbd := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	if err != nil {
		panic(err)
	}

	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rbd,
	}, logger)
	if err != nil {
		panic(err)
	}

	subPlayerJoined, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rbd,
		ConsumerGroup: "PlayerJoined",
	}, logger)

	subTeamCreated, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rbd,
		ConsumerGroup: "TeamCreated",
	}, logger)

	client := ScoreboardAPIClient{}

	var lock sync.Mutex
	players := map[string]Player{}
	teams := map[string]Team{}

	router.AddMiddleware(middleware.CorrelationID)

	router.AddHandler(
		"OnPlayerJoined",
		"player_joined",
		subPlayerJoined,
		"team_created",
		pub,
		func(msg *message.Message) ([]*message.Message, error) {
			var event PlayerJoined
			err := json.Unmarshal(msg.Payload, &event)
			if err != nil {
				return nil, err
			}

			lock.Lock()
			defer lock.Unlock()

			player := players[event.ID]

			for key, team := range teams {
				if team.Name == player.Team {
					team.Players = append(team.Players, player)
					teams[key] = team
					// Team found, no need to continue
					return nil, nil
				}
			}

			// Team not found, create it
			team := Team{
				ID:      uuid.NewString(),
				Name:    player.Team,
				Players: []Player{player},
			}

			teams[team.ID] = team

			teamCreatedEvent := TeamCreated{
				ID: team.ID,
			}

			var correlationID string
			correlationID = middleware.MessageCorrelationID(msg)

			if correlationID == "" {
				correlationID = uuid.NewString()
			}

			payload, err := json.Marshal(teamCreatedEvent)
			if err != nil {
				return nil, err
			}

			newMessage := message.NewMessage(uuid.NewString(), payload)
			middleware.SetCorrelationID(correlationID, msg)

			return []*message.Message{newMessage}, nil
		},
	)

	router.AddNoPublisherHandler(
		"OnTeamCreated",
		"team_created",
		subTeamCreated,
		func(msg *message.Message) error {
			var event TeamCreated
			err := json.Unmarshal(msg.Payload, &event)
			if err != nil {
				return err
			}

			// TODO
			correlationID := middleware.MessageCorrelationID(msg)

			err = client.CreateTeamScoreboard(event.ID, correlationID)
			if err != nil {
				return err
			}

			return nil
		},
	)

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	go func() {
		err := router.Run(ctx)
		if err != nil {
			panic(err)
		}
	}()

	<-router.Running()

	e := echo.New()
	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "ok")
	})

	e.POST("/players", func(c echo.Context) error {
		var request PlayerJoinRequest
		if err := c.Bind(&request); err != nil {
			return err
		}

		player := Player{
			ID:   uuid.NewString(),
			Name: request.Name,
			Team: request.Team,
		}

		lock.Lock()
		players[player.ID] = player
		lock.Unlock()

		event := PlayerJoined{
			ID: player.ID,
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		correlationID := c.Request().Header.Get("Correlation-ID")

		msg := message.NewMessage(uuid.NewString(), payload)
		middleware.SetCorrelationID(correlationID, msg)
		err = pub.Publish("player_joined", msg)
		if err != nil {
			return err
		}

		return nil
	})

	err = e.Start(":8080")
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

type ScoreboardAPIClient struct{}

func (c ScoreboardAPIClient) CreateTeamScoreboard(teamID string, correlationID string) error {
	clients, err := clients.NewClients(os.Getenv("GATEWAY_ADDR"), nil)
	if err != nil {
		return err
	}

	resp, err := clients.Scoreboard.PostTeamsWithResponse(
		context.Background(),
		&scoreboard.PostTeamsParams{
			CorrelationID: correlationID,
		},
		scoreboard.AddTeamScoreboardRequest{
			Team: teamID,
		},
	)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}
