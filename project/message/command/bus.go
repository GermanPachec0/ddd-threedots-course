package command

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewCommandBus(pub message.Publisher) *cqrs.CommandBus {
	commandBus, err := cqrs.NewCommandBusWithConfig(
		pub,
		cqrs.CommandBusConfig{
			GeneratePublishTopic: func(cbgptp cqrs.CommandBusGeneratePublishTopicParams) (string, error) {
				return fmt.Sprintf("commands.%s", cbgptp.CommandName), nil
			},

			Marshaler: marshaler,
		},
	)
	if err != nil {
		panic(err)
	}

	return commandBus
}
