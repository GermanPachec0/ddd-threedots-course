package event

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewBus(pub message.Publisher) *cqrs.EventBus {
	eventBus, err := cqrs.NewEventBusWithConfig(
		pub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(geptp cqrs.GenerateEventPublishTopicParams) (string, error) {
				return fmt.Sprintf("events.%s", geptp.EventName), nil
			},
			Marshaler: marshaler,
		},
	)
	if err != nil {
		panic(err)
	}

	return eventBus
}
