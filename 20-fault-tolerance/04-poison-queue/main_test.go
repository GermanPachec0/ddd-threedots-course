// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

func Test(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer: 100,
		Persistent:          true,
	}, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poisonedMessages, err := pubSub.Subscribe(ctx, "PoisonQueue")
	if err != nil {
		t.Fatal(err)
	}

	storage := &testOrdersStorage{
		orders: map[string]string{},
	}

	err = ProcessMessages(ctx, pubSub, pubSub, storage)
	if err != nil {
		t.Error(err)
	}

	var expectedMessages, expectedPoisonedMessages []testMessage

	bus, err := cqrs.NewEventBusWithConfig(
		pubSub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: cqrs.JSONMarshaler{},
			Logger:    logger,
		},
	)

	for i := 0; i < 20; i++ {
		id := uuid.NewString()
		var link string

		if i%3 == 0 {
			expectedPoisonedMessages = append(expectedPoisonedMessages, testMessage{
				OrderID: id,
			})
		} else {
			link = "https://track.example.com/" + id
			expectedMessages = append(expectedMessages, testMessage{
				OrderID:      id,
				TrackingLink: link,
			})
		}

		event := OrderDispatched{
			OrderID:      id,
			TrackingLink: link,
		}

		err = bus.Publish(ctx, event)
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 10; i++ {
		if len(storage.orders) == len(expectedMessages) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if len(storage.orders) != len(expectedMessages) {
		t.Fatalf("expected %d orders, got %d", len(expectedMessages), len(storage.orders))
	}

	for _, expectedMessage := range expectedMessages {
		if storage.orders[expectedMessage.OrderID] != expectedMessage.TrackingLink {
			t.Errorf("expected tracking link %s for order %s, got %s", expectedMessage.TrackingLink, expectedMessage.OrderID, storage.orders[expectedMessage.OrderID])
		}
	}

	var actualPoisonedMessages []OrderDispatched
	for i := 0; i < len(expectedPoisonedMessages); i++ {
		select {
		case msg := <-poisonedMessages:
			var event OrderDispatched
			err := json.Unmarshal(msg.Payload, &event)
			if err != nil {
				t.Fatal(err)
			}

			actualPoisonedMessages = append(actualPoisonedMessages, event)

			msg.Ack()
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for poisoned message #%v", i+1)
		}
	}

	if len(actualPoisonedMessages) != len(expectedPoisonedMessages) {
		t.Fatalf("expected %d poisoned messages, got %d", len(expectedPoisonedMessages), len(actualPoisonedMessages))
	}

	for _, expectedMessage := range expectedPoisonedMessages {
		found := false

		for _, actualMessage := range actualPoisonedMessages {
			if actualMessage.OrderID == expectedMessage.OrderID {
				if actualMessage.TrackingLink != "" {
					t.Errorf("expected empty tracking link for order %s, got %s", expectedMessage.OrderID, actualMessage.TrackingLink)
				}

				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected poisoned message for order %s", expectedMessage.OrderID)
		}
	}
}

type testMessage struct {
	OrderID      string
	TrackingLink string
}

type testOrdersStorage struct {
	orders  map[string]string
	counter int
}

func (t *testOrdersStorage) AddTrackingLink(ctx context.Context, orderID string, trackingLink string) error {
	t.counter++

	if t.counter%5 == 0 {
		return errors.New("the database is down")
	}

	t.orders[orderID] = trackingLink

	return nil
}
