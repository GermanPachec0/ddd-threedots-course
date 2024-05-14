package main

import (
	"context"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

type NotificationsClient interface {
	SendOrderConfirmation(orderID string) error
}

type SpreadsheetsClient interface {
	AppendOrderRow(orderID string) error
}

func Subscribe(
	notificationsClient NotificationsClient,
	spreadsheetsClient SpreadsheetsClient,
) error {
	logger := watermill.NewStdLogger(false, false)

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	orderConfirmationSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "notifications",
	}, logger)
	if err != nil {
		return err
	}

	spreadsheetSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "spreadsheets",
	}, logger)
	if err != nil {
		return err
	}
	go processMessages(orderConfirmationSub, notificationsClient.SendOrderConfirmation)
	go processMessages(spreadsheetSub, spreadsheetsClient.AppendOrderRow)

	return nil
}

func processMessages(sub message.Subscriber, action func(orderID string) error) {
	messages, err := sub.Subscribe(context.Background(), "orders-placed")
	if err != nil {
		panic(err)
	}

	for msg := range messages {
		orderID := string(msg.Payload)

		err := action(orderID)
		if err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	}
}
