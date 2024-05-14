// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	type header struct {
		ID         string `json:"id"`
		EventName  string `json:"event_name"`
		OccurredAt string `json:"occurred_at"`
	}

	type productOutOfStock struct {
		Header    header `json:"header"`
		ProductID string `json:"product_id"`
	}

	type productBackInStock struct {
		Header    header `json:"header"`
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}

	pubSub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer: 10,
	}, watermill.NopLogger{})

	messages, err := pubSub.Subscribe(context.Background(), "product-updates")
	require.NoError(t, err)

	publisher := NewPublisher(pubSub)

	productID := uuid.NewString()

	err = publisher.PublishProductOutOfStock(productID)
	require.NoError(t, err)

	select {
	case msg := <-messages:
		fmt.Println(string(msg.Payload))
		var event productOutOfStock
		err := json.Unmarshal(msg.Payload, &event)
		require.NoError(t, err)

		assert.NotEmpty(t, event.Header.ID)
		assert.Equal(t, "ProductOutOfStock", event.Header.EventName)
		assert.NotEmpty(t, event.Header.OccurredAt)
		assert.Equal(t, productID, event.ProductID)

		msg.Ack()
	case <-time.After(time.Second):
		t.Fatal("no message received")
	}

	err = publisher.PublishProductBackInStock(productID, 10)
	require.NoError(t, err)

	select {
	case msg := <-messages:
		var event productBackInStock
		err := json.Unmarshal(msg.Payload, &event)
		require.NoError(t, err)

		assert.NotEmpty(t, event.Header.ID)
		assert.Equal(t, "ProductBackInStock", event.Header.EventName)
		assert.NotEmpty(t, event.Header.OccurredAt)
		assert.Equal(t, productID, event.ProductID)
		assert.Equal(t, 10, event.Quantity)

		msg.Ack()
	case <-time.After(time.Second):
		t.Fatal("no message received")
	}
}
