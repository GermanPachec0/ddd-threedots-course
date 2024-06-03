package event

import (
	"context"
	"fmt"
	"tickets/entities"
)

func (h *Handler) BookTicketToDeadNotion(ctx context.Context, event *entities.BookingMade) error {
	show, err := h.showRepo.ShowByID(ctx, event.ShowId)
	if err != nil {
		return fmt.Errorf("failed to get show: %w", err)
	}
	err = h.deadNationSvc.CreateBooking(ctx, entities.DeadNationBookingRequest{
		CustomerEmail:     event.CustomerEmail,
		DeadNationEventID: show.DeadNationID,
		NumberOfTickets:   event.NumberOfTickets,
		BookingID:         event.BookingID,
	})
	if err != nil {
		return fmt.Errorf("failed to book in dead nation: %w", err)
	}

	return nil
}
