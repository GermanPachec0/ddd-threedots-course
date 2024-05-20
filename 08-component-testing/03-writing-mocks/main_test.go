// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceiptsServiceMock(t *testing.T) {
	mock := ReceiptsServiceMock{}

	req1 := IssueReceiptRequest{
		TicketID: uuid.NewString(),
		Price: Money{
			Amount:   fmt.Sprintf("%d", rand.Intn(10)+1),
			Currency: "EUR",
		},
	}

	response, err := mock.IssueReceipt(context.Background(), req1)
	require.NoError(t, err)
	require.NotEmpty(t, response.ReceiptNumber)
	require.NotZero(t, response.IssuedAt)

	req2 := IssueReceiptRequest{
		TicketID: uuid.NewString(),
		Price: Money{
			Amount:   fmt.Sprintf("%d", rand.Intn(10)+1),
			Currency: "EUR",
		},
	}
	response, err = mock.IssueReceipt(context.Background(), req2)
	require.NoError(t, err)
	require.NotEmpty(t, response.ReceiptNumber)
	require.NotZero(t, response.IssuedAt)

	assert.Equal(t, []IssueReceiptRequest{req1, req2}, mock.IssuedReceipts)
}
