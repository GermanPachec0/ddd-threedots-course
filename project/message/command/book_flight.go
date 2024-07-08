package command

import (
	"context"
	"errors"
	"fmt"
	"tickets/entities"
)

func (h Handler) BookFlight(ctx context.Context, command *entities.BookFlight) error {
	resp, err := h.transportaionService.BookFlight(ctx, entities.BookFlightTicketRequest{
		CustomerEmail:  command.CustomerEmail,
		FlightID:       command.FlightID,
		PassengerNames: command.Passengers,
		ReferenceId:    command.ReferenceID,
		IdempotencyKey: command.IdempotencyKey,
	})
	if errors.Is(err, entities.ErrNoFlightTicketsAvailable) {
		err = h.eventBus.Publish(ctx, entities.FlightBookingFailed_v1{
			Header:        entities.NewEventHeader(),
			FailureReason: err.Error(),
			FlightID:      command.FlightID,
			ReferenceID:   command.ReferenceID,
		})
		if err != nil {
			return fmt.Errorf("failed to publish FlightBookingFailed_v1 event: %w", err)
		}

		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to void receipt: %w", err)
	}

	err = h.eventBus.Publish(ctx, entities.FlightBooked_v1{
		Header:      entities.NewEventHeader(),
		FlightID:    command.FlightID,
		TicketIDs:   resp.TicketIds,
		ReferenceID: command.ReferenceID,
	})
	if err != nil {
		return fmt.Errorf("failed to publish FlightBooked_v1 event: %w", err)
	}

	return nil
}
