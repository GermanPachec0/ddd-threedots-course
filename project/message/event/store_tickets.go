package event

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func (h Handler) StoreTickets(ctx context.Context, event *entities.TicketBookingConfirmed_v1) error {
	log.FromContext(ctx).Info("Storing ticket")

	return h.ticketRepo.Create(ctx, entities.Ticket{
		TicketID:      event.TicketID,
		Price:         event.Price,
		CustomerEmail: event.CustomerEmail,
	})
}
