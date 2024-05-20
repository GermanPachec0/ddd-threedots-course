package main

import (
	"context"
	"sync"
	"time"
)

type IssueReceiptRequest struct {
	TicketID string `json:"ticket_id"`
	Price    Money  `json:"price"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request IssueReceiptRequest) (IssueReceiptResponse, error)
}

type ReceiptsServiceMock struct {
	mock           sync.Mutex
	IssuedReceipts []IssueReceiptRequest
}

func (m *ReceiptsServiceMock) IssueReceipt(ctx context.Context, request IssueReceiptRequest) (IssueReceiptResponse, error) {
	m.mock.Lock()
	defer m.mock.Unlock()
	m.IssuedReceipts = append(m.IssuedReceipts, request)
	return IssueReceiptResponse{
		ReceiptNumber: request.TicketID,
		IssuedAt:      time.Now(),
	}, nil
}

func main() {}
