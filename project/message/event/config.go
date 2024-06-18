package event

import (
	"fmt"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

var marshaler = cqrs.JSONMarshaler{
	GenerateName: cqrs.StructName,
}

func NewProcessorConfig(redisClient *redis.Client, watermillLogger watermill.LoggerAdapter) cqrs.EventProcessorConfig {
	return cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			handlerEvent := params.EventHandler.NewEvent()
			event, ok := handlerEvent.(entities.IEvent)

			if !ok {
				return "", fmt.Errorf("invalid event type: %T doesn't implement entities.Event", handlerEvent)
			}
			var prefix string
			if event.IsInternal() {
				prefix = "internal-events.svc-tickets."
			} else {
				prefix = "events."
			}

			return fmt.Sprintf(prefix + params.EventName), nil
		},
		SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return redisstream.NewSubscriber(redisstream.SubscriberConfig{
				Client:        redisClient,
				ConsumerGroup: "svc-tickets.events." + params.HandlerName,
			}, watermillLogger)
		},
		Marshaler: marshaler,
		Logger:    watermillLogger,
	}
}
