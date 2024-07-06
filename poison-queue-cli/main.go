package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/urfave/cli/v2"
)

const PoisonQueueTopic = "PoisonQueue"

type Message struct {
	ID     string
	Reason string
}

type Handler struct {
	firstMessage string
	subscriber   message.Subscriber
	publisher    message.Publisher
}

func NewHandler() (*Handler, error) {
	logger := watermill.NewStdLogger(false, false)

	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	sub, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               []string{os.Getenv("KAFKA_ADDR")},
			Unmarshaler:           kafka.DefaultMarshaler{},
			ConsumerGroup:         "poison-queue-cli",
			OverwriteSaramaConfig: cfg,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	pub, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:   []string{os.Getenv("KAFKA_ADDR")},
			Marshaler: kafka.DefaultMarshaler{},
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	return &Handler{
		subscriber:   sub,
		publisher:    pub,
		firstMessage: "",
	}, nil
}
func (h *Handler) Preview(ctx context.Context) ([]Message, error) {
	var messages []Message

	router, err := message.NewRouter(
		message.RouterConfig{},
		watermill.NewStdLogger(false, false),
	)

	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)

	firstMessage := ""

	done := false

	router.AddHandler(
		"preview",
		PoisonQueueTopic,
		h.subscriber,
		PoisonQueueTopic,
		h.publisher,
		func(msg *message.Message) ([]*message.Message, error) {
			if done {
				cancel()
				return nil, errors.New("done")
			}

			if firstMessage == "" {
				firstMessage = msg.UUID
			} else if firstMessage == msg.UUID {
				done = true
				return nil, errors.New("done")
			}
			messages = append(messages,
				Message{
					ID:     msg.UUID,
					Reason: msg.Metadata.Get(middleware.ReasonForPoisonedKey)})
			return []*message.Message{msg}, nil
		})

	err = router.Run(ctx)
	if err != nil {
		return nil, err
	}
	return messages, nil

}
func (h *Handler) Remove(ctx context.Context, messageID string) error {
	router, err := message.NewRouter(
		message.RouterConfig{},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)

	done := false
	firstMessage := ""

	founded := false
	router.AddHandler(
		"remove",
		PoisonQueueTopic,
		h.subscriber,
		PoisonQueueTopic,
		h.publisher,
		func(msg *message.Message) ([]*message.Message, error) {
			if done {
				cancel()
				return nil, errors.New("done")
			}
			if messageID == msg.UUID {
				founded = true
				done = true
				msg.Ack()
				return nil, nil
			}
			if firstMessage == "" {
				firstMessage = msg.UUID
			} else if firstMessage == msg.UUID {
				done = true
				return nil, errors.New("done")
			}

			return []*message.Message{msg}, nil

		})

	err = router.Run(ctx)
	if err != nil {
		return err
	}

	if founded {
		return nil
	}

	return errors.New("message not found")
}

func (h *Handler) Requeue(ctx context.Context, messageID string) error {
	router, err := message.NewRouter(
		message.RouterConfig{},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)

	done := false
	firstMessage := ""

	founded := false
	router.AddHandler(
		"remove",
		PoisonQueueTopic,
		h.subscriber,
		PoisonQueueTopic,
		h.publisher,
		func(msg *message.Message) ([]*message.Message, error) {
			if done {
				cancel()
				return nil, errors.New("done")
			}
			if messageID == msg.UUID {
				founded = true
				done = true
				msg.Ack()
				topic := middleware.PoisonedTopicKey
				err := h.publisher.Publish(topic, msg)
				if err != nil {
					return nil, err
				}
				return nil, nil
			}
			if firstMessage == "" {
				firstMessage = msg.UUID
			} else if firstMessage == msg.UUID {
				done = true
				return nil, errors.New("done")
			}

			return []*message.Message{msg}, nil

		})

	err = router.Run(ctx)
	if err != nil {
		return err
	}

	if founded {
		return nil
	}

	return errors.New("message not found")
}

func main() {
	app := &cli.App{
		Name:  "poison-queue-cli",
		Usage: "Manage the Poison Queue",
		Commands: []*cli.Command{
			{
				Name:  "preview",
				Usage: "preview messages",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					messages, err := h.Preview(c.Context)
					if err != nil {
						return err
					}

					for _, m := range messages {
						fmt.Printf("%v\t%v\n", m.ID, m.Reason)
					}

					return nil
				},
			},
			{
				Name:      "remove",
				ArgsUsage: "<message_id>",
				Usage:     "remove message",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					err = h.Remove(c.Context, c.Args().First())
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:      "requeue",
				ArgsUsage: "<message_id>",
				Usage:     "requeue message",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					err = h.Requeue(c.Context, c.Args().First())
					if err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
