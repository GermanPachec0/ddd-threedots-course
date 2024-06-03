package main

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

type NotificationShouldBeSent struct {
	NotificationID string
	Email          string
	Message        string
}

type SendNotification struct {
	NotificationID string
	Email          string
	Message        string
}

type Sender interface {
	SendNotification(ctx context.Context, notificationID, email, message string) error
}

func NewProcessor(router *message.Router, sender Sender, sub message.Subscriber, watermillLogger watermill.LoggerAdapter) *cqrs.CommandProcessor {
	commandProcessor, err := cqrs.NewCommandProcessorWithConfig(
		router,
		cqrs.CommandProcessorConfig{
			GenerateSubscribeTopic: func(params cqrs.CommandProcessorGenerateSubscribeTopicParams) (string, error) {
				return "commands", nil
			},
			SubscriberConstructor: func(params cqrs.CommandProcessorSubscriberConstructorParams) (message.Subscriber, error) {
				return sub, nil
			},
			Marshaler: cqrs.JSONMarshaler{
				GenerateName: cqrs.StructName,
			},
			Logger: watermillLogger,
		},
	)
	if err != nil {
		panic(err)
	}

	err = commandProcessor.AddHandlers(cqrs.NewCommandHandler(
		"SendNotification",
		func(ctx context.Context, command *SendNotification) error {
			fmt.Println("Sending notification", command.NotificationID, command.Email, command.Message)
			return sender.SendNotification(ctx, command.NotificationID, command.Email, command.Message)
		},
	))
	if err != nil {
		panic(err)
	}

	return commandProcessor
}
