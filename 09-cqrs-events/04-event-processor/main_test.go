// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DocumentDownloaded struct {
	ID string `json:"id"`
}

type DocumentDownloadedHandler struct {
	HandledEvents chan DocumentDownloaded
}

func (h *DocumentDownloadedHandler) HandlerName() string {
	return "document_downloaded_handler"
}

func (h *DocumentDownloadedHandler) NewEvent() interface{} {
	return &DocumentDownloaded{}
}

func (h *DocumentDownloadedHandler) Handle(ctx context.Context, event any) error {
	e := event.(*DocumentDownloaded)

	fmt.Printf("Handling event %#v\n", e)

	h.HandledEvents <- *e

	return nil
}

func Test(t *testing.T) {
	logger := watermill.NewStdLogger(true, true)

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			Persistent: true,
		},
		logger,
	)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)

	marshaler := cqrs.JSONMarshaler{
		GenerateName: cqrs.StructName,
	}
	eventBus, err := cqrs.NewEventBusWithConfig(
		pubSub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: marshaler,
		},
	)
	require.NoError(t, err)

	event := DocumentDownloaded{ID: "1"}

	err = eventBus.Publish(context.Background(), event)
	require.NoError(t, err)

	handler := &DocumentDownloadedHandler{make(chan DocumentDownloaded, 1)}

	err = RegisterEventHandlers(
		pubSub,
		router,
		[]cqrs.EventHandler{handler},
		logger,
	)
	require.NoError(t, err)

	require.Len(
		t,
		router.Handlers(),
		1,
		"expected one handler to be registered in router, have you called AddHandler and AddHandlersToRouter on event processor?",
	)

	go func() {
		err := router.Run(context.Background())
		assert.NoError(t, err)
	}()

	<-router.Running()

	err = eventBus.Publish(context.Background(), event)
	require.NoError(t, err)

	select {
	case handledEvent := <-handler.HandledEvents:
		assert.Equal(t, event, handledEvent)
	case <-time.After(100 * time.Millisecond):
		require.Fail(t, "no DocumentDownloaded event received")
	}
}
