package main

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	_ "github.com/lib/pq"
)

func RunForwarder(
	db *sqlx.DB,
	rdb *redis.Client,
	outboxTopic string,
	logger watermill.LoggerAdapter,
) error {

	postgresSub, err := SubscribeForMessages(db, outboxTopic, logger)
	if err != nil {
		return err
	}

	redisPub := NewRedisPublisher(rdb, logger)

	fwd, err := forwarder.NewForwarder(postgresSub, redisPub, logger,
		forwarder.Config{
			ForwarderTopic: outboxTopic,
			Middlewares: []message.HandlerMiddleware{
				func(h message.HandlerFunc) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						fmt.Println("Forwarding message", msg.UUID, string(msg.Payload))
						return h(msg)
					}
				},
			},
		})

	if err != nil {
		return err
	}

	go func() {
		err := fwd.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	<-fwd.Running()

	return nil
}

func NewRedisPublisher(rdb *redis.Client, watermillLogger watermill.LoggerAdapter) message.Publisher {
	var pub message.Publisher
	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, watermillLogger)
	if err != nil {
		panic(err)
	}
	pub = log.CorrelationPublisherDecorator{pub}

	return pub
}
func SubscribeForMessages(db *sqlx.DB, topic string, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	subConfig := sql.SubscriberConfig{
		SchemaAdapter:    sql.DefaultPostgreSQLSchema{},
		OffsetsAdapter:   sql.DefaultPostgreSQLOffsetsAdapter{},
		InitializeSchema: true,
	}

	sub, err := sql.NewSubscriber(db, subConfig, logger)
	if err != nil {
		return nil, err
	}
	err = sub.SubscribeInitialize(topic)
	if err != nil {
		return nil, err
	}

	return sub, nil
}
