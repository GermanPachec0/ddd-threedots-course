// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type eventsCounter struct {
	count int
}

func (e *eventsCounter) CountEvent() error {
	e.count++
	return nil
}

func TestSomeEventHandler(t *testing.T) {
	counter := &eventsCounter{}
	handler := NewFollowRequestSentHandler(counter)

	require.NotEmpty(t, handler.HandlerName())
	require.IsType(t, &FollowRequestSent{}, handler.NewEvent())

	err := handler.Handle(context.Background(), &FollowRequestSent{
		From: uuid.NewString(),
		To:   uuid.NewString(),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, counter.count)

	err = handler.Handle(context.Background(), &FollowRequestSent{
		From: uuid.NewString(),
		To:   uuid.NewString(),
	})
	require.NoError(t, err)
	assert.Equal(t, 2, counter.count)
}
