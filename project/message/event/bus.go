package event

import (
	"fmt"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Event interface {
	IsInternal() bool
}

func NewBus(pub message.Publisher) *cqrs.EventBus {
	eventBus, err := cqrs.NewEventBusWithConfig(
		pub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(geptp cqrs.GenerateEventPublishTopicParams) (string, error) {
				event, ok := geptp.Event.(entities.IEvent)
				if !ok {
					return "", fmt.Errorf("invalid event type: %T doesn't implement entities.Event", geptp.Event)
				}

				if event.IsInternal() {
					return "internal-events.svc-tickets." + geptp.EventName, nil
				} else {
					return "events", nil

				}
			},
			Marshaler: marshaler,
		},
	)
	if err != nil {
		panic(err)
	}

	return eventBus
}
