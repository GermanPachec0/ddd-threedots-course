package event

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func (h Handler) TicketRefundToSheet(ctx context.Context, event *entities.TicketBookingCanceled_v1) error {
	log.FromContext(ctx).Info("Adding ticket refund to sheet")

	return h.spreadsheetsService.AppendRow(
		ctx,
		"tickets-to-refund",
		[]string{event.TicketID, event.CustomerEmail, event.Price.Amount, event.Price.Currency},
	)
}
