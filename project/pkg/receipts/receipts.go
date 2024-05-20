package receipts

import (
	"context"
	"fmt"
	"net/http"
	"tickets/adapters/echo_server/ticket"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"
)

type ReceiptsClient struct {
	clients *clients.Clients
}
type IssueReceiptRequest struct {
	TicketID string       `json:"ticket_id"`
	Price    ticket.Price `json:"price"`
}
type ReceiptService interface {
	IssueReceipt(ctx context.Context, ticketRequest IssueReceiptRequest) error
}

func NewReceiptsClient(clients *clients.Clients) ReceiptsClient {
	return ReceiptsClient{
		clients: clients,
	}
}

func (c ReceiptsClient) IssueReceipt(ctx context.Context, ticketRequest IssueReceiptRequest) error {

	body := receipts.PutReceiptsJSONRequestBody{
		TicketId: ticketRequest.TicketID,
		Price: receipts.Money{
			MoneyAmount:   ticketRequest.Price.Amount,
			MoneyCurrency: ticketRequest.Price.Currency,
		},
	}
	receiptsResp, err := c.clients.Receipts.PutReceiptsWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if receiptsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", receiptsResp.StatusCode())
	}

	return nil
}
