package event

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func (h Handler) AppendToTracker(ctx context.Context, event *entities.TicketBookingConfirmed_v1) error {
	log.FromContext(ctx).Info("Generating ticket for booking")

	return h.spreadsheetsService.AppendRow(ctx, "tickets-to-print", []string{event.TicketID, event.CustomerEmail, event.Price.Amount, event.Price.Currency})
}
