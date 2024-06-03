package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/payments"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"
)

type ReceiptsServiceClient struct {
	clients *clients.Clients
}

func NewReceiptsServiceClient(clients *clients.Clients) *ReceiptsServiceClient {
	if clients == nil {
		panic("Clients receipts service is nil")
	}
	return &ReceiptsServiceClient{clients: clients}
}

func (c ReceiptsServiceClient) RefundPayment(ctx context.Context, cmd entities.RefundTicket) error {
	resp, err := c.clients.Payments.PutRefundsWithResponse(ctx,
		payments.PaymentRefundRequest{
			// we are using TicketID as a payment reference
			PaymentReference: cmd.TicketID,
			Reason:           "customer requested refund",
			DeduplicationId:  &cmd.Header.IdempotencyKey,
		})

	if err != nil {
		return fmt.Errorf("Error refunding payments  %w", err)
	}

	slog.Info("response %w", resp)
	return nil
}

func (c ReceiptsServiceClient) RefundVoidReceipts(ctx context.Context, cmd entities.RefundTicket) error {
	resp, err := c.clients.Receipts.PutVoidReceiptWithResponse(ctx,
		receipts.VoidReceiptRequest{
			Reason:       "customer requested refund",
			IdempotentId: &cmd.Header.IdempotencyKey,
			TicketId:     cmd.TicketID,
		})

	if err != nil {
		return fmt.Errorf("Error refunding void receipt %w", err)
	}

	slog.Info("response %w", resp)
	return nil
}

func (c ReceiptsServiceClient) IssueReceipt(ctx context.Context, request entities.IssueReceiptRequest) (entities.IssueReceiptResponse, error) {
	resp, err := c.clients.Receipts.PutReceiptsWithResponse(ctx, receipts.CreateReceipt{
		IdempotencyKey: &request.IdempotencyKey,
		Price: receipts.Money{
			MoneyAmount:   request.Price.Amount,
			MoneyCurrency: request.Price.Currency,
		},
		TicketId: request.TicketID,
	})
	if err != nil {
		return entities.IssueReceiptResponse{}, fmt.Errorf("Failed to post receipt")
	}
	switch resp.StatusCode() {
	case http.StatusOK:
		// receipt already exists
		return entities.IssueReceiptResponse{
			ReceiptNumber: resp.JSON200.Number,
			IssuedAt:      resp.JSON200.IssuedAt,
		}, nil
	case http.StatusCreated:
		// receipt was created
		return entities.IssueReceiptResponse{
			ReceiptNumber: resp.JSON201.Number,
			IssuedAt:      resp.JSON201.IssuedAt,
		}, nil
	default:
		return entities.IssueReceiptResponse{}, fmt.Errorf("unexpected status code for POST receipts-api/receipts: %d", resp.StatusCode())
	}
}
