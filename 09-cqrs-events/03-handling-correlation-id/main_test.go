// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockPublisher struct {
	PublishedMessages []*message.Message
}

func (m *MockPublisher) Publish(topic string, messages ...*message.Message) error {
	m.PublishedMessages = append(m.PublishedMessages, messages...)
	return nil
}

func (m *MockPublisher) Close() error {
	return nil
}

func TestCorrelationPublisherDecorator(t *testing.T) {
	mockPublisher := &MockPublisher{}

	var publisher message.Publisher = mockPublisher
	publisher = CorrelationPublisherDecorator{publisher}

	msg := message.NewMessage(watermill.NewUUID(), nil)
	expecedCorrelationID := uuid.NewString()
	msg.SetContext(ContextWithCorrelationID(context.Background(), expecedCorrelationID))

	err := publisher.Publish("test", msg)
	require.NoError(t, err)

	require.Equal(
		t,
		1,
		len(mockPublisher.PublishedMessages),
		"one message should be published",
	)

	assert.Equal(
		t,
		expecedCorrelationID,
		mockPublisher.PublishedMessages[0].Metadata.Get("correlation_id"),
		"correlation_id should be set in metadata",
	)
}
