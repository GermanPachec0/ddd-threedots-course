package api

import (
	"fmt"
	"net/http"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/payments"
	"golang.org/x/net/context"
)

type PaymentsServiceClient struct {
	// we are not mocking this client: it's pointless to use interface here
	clients *clients.Clients
}

func NewPaymentsServiceClient(clients *clients.Clients) PaymentsServiceClient {
	if clients == nil {
		panic("NewPaymentsServiceClient: clients is nil")
	}

	return PaymentsServiceClient{clients: clients}
}

func (c PaymentsServiceClient) RefundPayment(ctx context.Context, refundPayment entities.PaymentRefund) error {
	resp, err := c.clients.Payments.PutRefundsWithResponse(ctx, payments.PaymentRefundRequest{
		PaymentReference: refundPayment.TicketID,
		Reason:           refundPayment.RefundReason,
		DeduplicationId:  &refundPayment.IdempotencyKey,
	})
	if err != nil {
		return fmt.Errorf("failed to post refund for payment %s: %w", refundPayment.TicketID, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected for /payments-api/refunds status code: %d", resp.StatusCode())
	}

	return nil
}
