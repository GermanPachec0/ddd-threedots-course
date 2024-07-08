package event

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func (h Handler) DeleteTicketCancel(ctx context.Context, event *entities.TicketBookingCanceled_v1) error {
	log.FromContext(ctx).Info("Removing cancel ticket")

	return h.ticketRepo.Delete(ctx, entities.Ticket{
		TicketID:      event.TicketID,
		Price:         event.Price,
		CustomerEmail: event.CustomerEmail,
		DeleteAt:      &event.Header.PublishedAt,
	})
}
