package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/Shopify/sarama"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

type AlertTriggered struct {
	AlertID     string    `json:"alert_id"`
	TriggeredAt time.Time `json:"triggered_at"`
}

type AlertResolved struct {
	AlertID    string    `json:"alert_id"`
	ResolvedAt time.Time `json:"resolved_at"`
}

type AlertUpdated struct {
	AlertID         string    `json:"alert_id"`
	IsTriggered     bool      `json:"is_triggered"`
	LastTriggeredAt time.Time `json:"last_triggered_at"`
	LastResolvedAt  time.Time `json:"last_resolved_at"`
}

func main() {
	logger := watermill.NewStdLogger(false, false)

	kafkaAddr := os.Getenv("KAFKA_ADDR")

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

	eventProcessor, err := cqrs.NewEventProcessorWithConfig(
		router,
		cqrs.EventProcessorConfig{
			GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
				return params.EventName, nil
			},
			SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
				sub, err := kafka.NewSubscriber(kafka.SubscriberConfig{
					Brokers:       []string{kafkaAddr},
					Unmarshaler:   kafka.DefaultMarshaler{},
					ConsumerGroup: params.HandlerName,
					// Make sure to use this config: it lets us validate your solution!
					OverwriteSaramaConfig: newConfig(),
				}, logger)
				if err != nil {
					panic(err)
				}
				return sub, nil
			},
			AckOnUnknownEvent: true,
			Marshaler:         cqrs.JSONMarshaler{},
			Logger:            logger,
		},
	)
	if err != nil {
		panic(err)
	}

	pub, err := kafka.NewPublisher(kafka.PublisherConfig{
		Brokers:   []string{kafkaAddr},
		Marshaler: kafka.DefaultMarshaler{},
	}, logger)
	if err != nil {
		panic(err)
	}

	eventBus, err := cqrs.NewEventBusWithConfig(
		pub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: cqrs.JSONMarshaler{},
			Logger:    logger,
		},
	)
	if err != nil {
		panic(err)
	}

	lock := sync.Mutex{}
	alerts := map[string]AlertUpdated{}

	err = eventProcessor.AddHandlers(
		cqrs.NewEventHandler("OnAlertTriggered", func(ctx context.Context, event *AlertTriggered) error {
			lock.Lock()
			defer lock.Unlock()

			alert, ok := alerts[event.AlertID]
			if !ok {
				alert = AlertUpdated{
					AlertID: event.AlertID,
				}
			}

			alert.LastTriggeredAt = event.TriggeredAt
			alert.IsTriggered = true
			alerts[event.AlertID] = alert

			return eventBus.Publish(ctx, alert)
		}),
		cqrs.NewEventHandler("OnAlertResolved", func(ctx context.Context, event *AlertResolved) error {
			lock.Lock()
			defer lock.Unlock()

			alert, ok := alerts[event.AlertID]
			if !ok {
				alert = AlertUpdated{
					AlertID: event.AlertID,
				}
			}
			alert.LastResolvedAt = event.ResolvedAt
			alert.IsTriggered = false
			alerts[event.AlertID] = alert

			return eventBus.Publish(ctx, alert)
		}),
	)
	if err != nil {
		panic(err)
	}

	err = router.Run(context.Background())
	if err != nil {
		panic(err)
	}
}

func newConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	return cfg
}
