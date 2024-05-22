// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DocumentDownloaded struct {
	ID string `json:"id"`
}

func Test(t *testing.T) {
	logger := watermill.NewStdLogger(true, true)

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, logger)
	if err != nil {
		panic(err)
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)

	marshaler := cqrs.JSONMarshaler{
		GenerateName: cqrs.StructName,
	}
	eventBus, err := cqrs.NewEventBusWithConfig(
		pub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: marshaler,
		},
	)
	require.NoError(t, err)

	event := DocumentDownloaded{ID: "bar"}

	go func() {
		for {
			err = eventBus.Publish(context.Background(), event)
			require.NoError(t, err)
			time.Sleep(time.Millisecond * 100)
		}
	}()

	processor, err := NewEventProcessor(router, rdb, marshaler, logger)
	require.NoError(t, err)

	handler1Events := make(chan DocumentDownloaded, 1)
	err = processor.AddHandlers(cqrs.NewEventHandler(
		"handler_1",
		func(ctx context.Context, event *DocumentDownloaded) error {
			handler1Events <- *event
			return nil
		},
	))
	require.NoError(t, err)

	handler2Events := make(chan DocumentDownloaded, 1)
	err = processor.AddHandlers(cqrs.NewEventHandler(
		"handler_2",
		func(ctx context.Context, event *DocumentDownloaded) error {
			handler2Events <- *event
			return nil
		},
	))
	require.NoError(t, err)

	go func() {
		err := router.Run(context.Background())
		assert.NoError(t, err)
	}()

	<-router.Running()

	err = eventBus.Publish(context.Background(), event)
	require.NoError(t, err)

	handlerReceivedNum := 0

	select {
	case <-handler1Events:
		t.Log("handler 1 received an event")
		handlerReceivedNum = 1
	case <-handler2Events:
		t.Log("handler 2 received an event")
		handlerReceivedNum = 2
	case <-time.After(1 * time.Second):
		t.Fatal("handler 1 didn't receive an event")
	}

	if handlerReceivedNum == 1 {
		select {
		case <-handler2Events:
			t.Log("handler 2 received an event")
		case <-time.After(1 * time.Second):
			t.Fatal("handler 2 didn't receive an event, did you define a consumer group name based on the handler name? Please check in logs if the consumer_group is different for each handler")
		}
	} else {
		select {
		case <-handler1Events:
			t.Log("handler 1 received an event")
		case <-time.After(1 * time.Second):
			t.Fatal("handler 1 didn't receive an event, did you define a consumer group name based on the handler name? Please check in logs if the consumer_group is different for each handler")
		}
	}
}
