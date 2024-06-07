package api

import (
	"context"
	"sync"
	"tickets/entities"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type ReceiptsMock struct {
	mock           sync.Mutex
	IssuedReceipts []entities.IssueReceiptRequest
}

func (c *ReceiptsMock) IssueReceipt(ctx context.Context, request entities.IssueReceiptRequest) (entities.IssueReceiptResponse, error) {
	c.mock.Lock()
	defer c.mock.Lock()

	c.IssuedReceipts = append(c.IssuedReceipts, request)
	return entities.IssueReceiptResponse{
		ReceiptNumber: "mocked-receipt-number",
		IssuedAt:      time.Now(),
	}, nil
}

func (c *ReceiptsMock) RefundPayment(ctx context.Context, cmd entities.TicketRefunded) error {
	c.mock.Lock()
	defer c.mock.Lock()

	return nil
}

func (c *ReceiptsMock) RefundVoidReceipts(ctx context.Context, cmd entities.TicketRefunded) error {
	c.mock.Lock()
	defer c.mock.Lock()

	return nil
}

func (c *ReceiptsMock) SetEventBus(eventBus *cqrs.EventBus) {
	c.mock.Lock()
	defer c.mock.Lock()

	return
}
