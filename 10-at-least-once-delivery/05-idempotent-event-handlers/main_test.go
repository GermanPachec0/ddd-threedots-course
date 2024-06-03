// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentTaken(t *testing.T) {
	repo := NewPaymentsRepository()
	handler := NewPaymentsHandler(repo)

	event0 := &PaymentTaken{
		PaymentID: uuid.NewString(),
		Amount:    100,
	}
	err := handler.HandlePaymentTaken(nil, event0)
	require.NoError(t, err)

	assert.Len(t, repo.Payments(), 1)
	assert.Equal(t, event0, &repo.Payments()[0])

	event1 := &PaymentTaken{
		PaymentID: uuid.NewString(),
		Amount:    200,
	}

	err = handler.HandlePaymentTaken(nil, event1)
	require.NoError(t, err)

	assert.Len(t, repo.Payments(), 2)
	assert.Equal(t, event0, &repo.Payments()[0])

	// test re-processing

	err = handler.HandlePaymentTaken(nil, event0)
	require.NoError(t, err)
	assert.Len(t, repo.Payments(), 2)

	err = handler.HandlePaymentTaken(nil, event1)
	require.NoError(t, err)
	assert.Len(t, repo.Payments(), 2)
}
