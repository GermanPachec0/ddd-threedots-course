// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIdempotency(t *testing.T) {
	mockHttp := &MockRoundTripper{}

	client := PalPayClient{mockHttp}

	idempotencyKey := uuid.NewString()

	err := client.Charge(ChargeRequest{
		CreditCardNumber: "1234-1234-1234-1234",
		CVV2:             "123",
		Amount:           1000,
		IdempotencyKey:   idempotencyKey,
	})
	require.NoError(t, err)

	sentIdempotencyKey := mockHttp.requests[0].Header.Get("Idempotency-Key")
	require.Equal(t, idempotencyKey, sentIdempotencyKey)
}

type MockRoundTripper struct {
	requests []*http.Request
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)

	resp := &http.Response{
		StatusCode: http.StatusOK,
	}

	return resp, nil
}
