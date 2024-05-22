package events

import (
	"encoding/json"
	"tickets/adapters/echo_server/ticket"
	"tickets/pkg/receipts"
	"tickets/pkg/sheets"

	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	brokenMessageID = "2beaf5bc-d5e4-4653-b075-2b36bbf28949"
	USDCurrency     = "USD"
)

type Handler struct {
	spreedShetClient sheets.SpreedSheetService
	receiptClient    receipts.ReceiptService
}

func NewHandler(spreedShetClient sheets.SpreedSheetService, receiptClient receipts.ReceiptService) Handler {
	return Handler{
		spreedShetClient,
		receiptClient,
	}
}

func (h *Handler) HandleTicketConfirmed(msg *message.Message) error {
	if string(msg.UUID) == brokenMessageID {
		return nil
	}

	if msg.Metadata.Get("type") != "TicketBookingConfirmed" {
		return nil
	}

	var ticket ticket.Ticket

	err := json.Unmarshal(msg.Payload, &ticket)
	if err != nil {
		return err
	}

	if ticket.Price.Currency == "" {
		ticket.Price.Currency = USDCurrency
	}

	sheetPayload := sheets.AppendToTrackerPayload{
		CustomerEmail: ticket.CustomerEmail,
		TicketID:      ticket.ID,
		Price:         ticket.Price,
	}

	ctx := msg.Context()

	err = h.spreedShetClient.AppendRow(ctx, "tickets-to-print", []string{sheetPayload.TicketID, sheetPayload.CustomerEmail, sheetPayload.Price.Amount, sheetPayload.Price.Currency})
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) HandleTicketCancel(msg *message.Message) error {
	if string(msg.UUID) == brokenMessageID {
		return nil
	}

	var ticket ticket.Ticket

	err := json.Unmarshal(msg.Payload, &ticket)
	if err != nil {
		return err
	}

	if ticket.Price.Currency == "" {
		ticket.Price.Currency = USDCurrency
	}

	sheetPayload := sheets.AppendToTrackerPayload{
		CustomerEmail: ticket.CustomerEmail,
		TicketID:      ticket.ID,
		Price:         ticket.Price,
	}

	ctx := msg.Context()

	err = h.spreedShetClient.AppendRow(ctx, "tickets-to-refund", []string{sheetPayload.TicketID, sheetPayload.CustomerEmail, sheetPayload.Price.Amount, sheetPayload.Price.Currency})
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) HandleTicketIssued(msg *message.Message) error {
	if string(msg.UUID) == brokenMessageID {
		return nil
	}

	if msg.Metadata.Get("type") != "TicketBookingConfirmed" {
		return nil
	}

	var ticket ticket.Ticket
	err := json.Unmarshal(msg.Payload, &ticket)
	if err != nil {
		return err
	}

	if ticket.Price.Currency == "" {
		ticket.Price.Currency = USDCurrency
	}

	issPayload := receipts.IssueReceiptRequest{
		TicketID: ticket.ID,
		Price:    ticket.Price,
	}
	ctx := msg.Context()
	err = h.receiptClient.IssueReceipt(ctx, issPayload)
	if err != nil {
		return err
	}
	return nil
}
