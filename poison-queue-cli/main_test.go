// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func Test(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	sub, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:     []string{os.Getenv("KAFKA_ADDR")},
			Unmarshaler: kafka.DefaultMarshaler{},
			InitializeTopicDetails: &sarama.TopicDetail{
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
		},
		logger,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = sub.SubscribeInitialize(PoisonQueueTopic)
	if err != nil {
		t.Fatal(err)
	}

	pub, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:   []string{os.Getenv("KAFKA_ADDR")},
			Marshaler: kafka.DefaultMarshaler{},
		},
		logger,
	)
	if err != nil {
		t.Fatal(err)
	}

	originalTopic := uuid.NewString()
	originalMessages, err := sub.Subscribe(context.Background(), originalTopic)
	if err != nil {
		t.Fatal(err)
	}

	var uuids []string
	for i := 0; i < 10; i++ {
		msg := message.NewMessage(watermill.NewUUID(), []byte("{}"))
		msg.Metadata.Set(middleware.ReasonForPoisonedKey, "network down")
		msg.Metadata.Set(middleware.PoisonedTopicKey, originalTopic)
		if err := pub.Publish(PoisonQueueTopic, msg); err != nil {
			t.Fatal(err)
		}
		uuids = append(uuids, msg.UUID)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}

	expectedUUIDs := []string{
		uuids[0],
		uuids[1],
		uuids[2],
		uuids[3],
		uuids[4],
		uuids[5],
		uuids[6],
		uuids[7],
		uuids[8],
		uuids[9],
	}

	assertMessages(t, expectedUUIDs)

	requeue(t, uuids[1])
	requeue(t, uuids[7])

	h, err = NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	err = h.Requeue(context.Background(), uuid.NewString())
	if err == nil {
		t.Fatal("expected to fail when requeuing unknown message ID")
	}

	expectedUUIDs = []string{
		uuids[0],
		uuids[2],
		uuids[3],
		uuids[4],
		uuids[5],
		uuids[6],
		uuids[8],
		uuids[9],
	}

	assertMessages(t, expectedUUIDs)

	for _, expectedUUID := range []string{uuids[1], uuids[7]} {
		select {
		case msg := <-originalMessages:
			if msg.UUID != expectedUUID {
				t.Fatalf("expected message with uuid %s, got %s", expectedUUID, msg.UUID)
			}
			msg.Ack()
		case <-time.After(time.Second):
		}
	}
}

func assertMessages(t *testing.T, expectedUUIDs []string) {
	t.Helper()

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}

	messages, err := h.Preview(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != len(expectedUUIDs) {
		t.Fatalf("expected %v messages, got %d", len(expectedUUIDs), len(messages))
	}

	for _, uuid := range expectedUUIDs {
		found := false
		for _, msg := range messages {
			if msg.ID == uuid {
				if msg.Reason != "network down" {
					t.Fatalf("expected reason to be 'network down', got %s", msg.Reason)
				}
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("expected message with uuid %s, but not found", uuid)
		}
	}
}

func requeue(t *testing.T, id string) {
	t.Helper()

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}

	err = h.Requeue(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
}
