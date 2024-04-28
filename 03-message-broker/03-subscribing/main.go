package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := watermill.NewStdLogger(false, false)

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client: rdb,
	}, logger)
	if err != nil {
		panic(err)
	}
	messages, err := subscriber.Subscribe(ctx, "progress")
	if err != nil {
		panic(err)
	}
	// 	Message ID: a16b6ab0-8c29-48f5-9d26-b508906af976 - 50%

	for msg := range messages {
		progress := string(msg.Payload)
		fmt.Printf("Message ID: %v - %v%%", msg.UUID, progress)
		msg.Ack()
	}
}
