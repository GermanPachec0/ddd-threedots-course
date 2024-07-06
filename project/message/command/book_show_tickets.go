package command

import (
	"context"
	"errors"
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

	return err
}
