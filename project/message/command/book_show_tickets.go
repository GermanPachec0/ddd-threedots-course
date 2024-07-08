package command

import (
	"context"
	"errors"
	"fmt"
	"tickets/db"
	"tickets/entities"
)

func (h Handler) BookShowTickets(ctx context.Context, command *entities.BookShowTickets) error {
	_, err := h.bookingsRepo.Create(ctx, entities.Booking{
		BookingID:       command.BookingID,
		ShowID:          command.ShowId,
		NumberOfTickets: command.NumberOfTickets,
		CustomerEmail:   command.CustomerEmail,
	})
	if errors.Is(err, db.ErrBookingAlreadyExists) {
		return nil
	}

	if errors.Is(err, db.ErrNoPlacesLeft) {
		publishErr := h.eventBus.Publish(ctx, entities.BookingFailed_v1{
			Header:        entities.NewEventHeader(),
			BookingID:     command.BookingID,
			FailureReason: err.Error(),
		})
		if publishErr != nil {
			return fmt.Errorf("failed to publish BookingFailed_v1 event: %w", publishErr)
		}
	}

	return err
}
