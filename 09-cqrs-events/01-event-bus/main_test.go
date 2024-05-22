// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DocumentPrinted struct {
	ID string `json:"id"`
}

type DocumentDownloaded struct {
	ID string `json:"id"`
}

func TestNewEventBus(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			Persistent: true,
		},
		logger,
	)

	eventBus, err := NewEventBus(pubSub)
	require.NoError(t, err)

	err = eventBus.Publish(context.Background(), DocumentPrinted{ID: "1"})
	require.NoError(t, err)

	err = eventBus.Publish(context.Background(), DocumentDownloaded{ID: "2"})
	require.NoError(t, err)

	someEvents, err := pubSub.Subscribe(context.Background(), "main.DocumentPrinted")
	require.NoError(t, err)

	anotherEvents, err := pubSub.Subscribe(context.Background(), "main.DocumentDownloaded")
	require.NoError(t, err)

	select {
	case msg := <-someEvents:
		assert.Equal(t, `{"id":"1"}`, string(msg.Payload))
	case <-time.After(100 * time.Millisecond):
		require.Fail(t, "no DocumentPrinted event received on main.DocumentPrinted topic")
	}

	select {
	case msg := <-anotherEvents:
		assert.Equal(t, `{"id":"2"}`, string(msg.Payload))
	case <-time.After(100 * time.Millisecond):
		require.Fail(t, "no DocumentDownloaded event received on main.DocumentDownloaded topic")
	}
}
