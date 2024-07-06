package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

type smsClient interface {
	SendSMS(phoneNumber string, message string) error
}

type UserSignedUp struct {
	Username    string `json:"username"`
	PhoneNumber string `json:"phone_number"`
	SignedUpAt  string `json:"signed_up_at"`
}

func ProcessMessages(
	ctx context.Context,
	sub message.Subscriber,
	smsClient smsClient,
) error {
	logger := watermill.NewStdLogger(false, false)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return err
	}
	t := middleware.NewThrottle(10, time.Second)
	router.AddMiddleware(t.Middleware)

	router.AddNoPublisherHandler(
		"send_welcome_message",
		"UserSignedUp",
		sub,
		func(msg *message.Message) error {
			event := UserSignedUp{}
			err := json.Unmarshal(msg.Payload, &event)
			if err != nil {
				return err
			}

			return smsClient.SendSMS(event.PhoneNumber, fmt.Sprintf("Welcome on board, %s!", event.Username))
		},
	)

	go func() {
		err := router.Run(ctx)
		if err != nil {
			panic(err)
		}
	}()

	<-router.Running()

	return nil
}
