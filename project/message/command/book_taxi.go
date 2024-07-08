package command

import (
	"context"
	"fmt"
	"tickets/entities"
)

func (h Handler) BookTaxi(ctx context.Context, command *entities.BookTaxi) error {

	resp, err := h.transportaionService.BookTaxi(ctx, entities.BookTaxi{
		CustomerEmail:      command.CustomerEmail,
		NumberOfPassengers: command.NumberOfPassengers,
		IdempotencyKey:     command.IdempotencyKey,
		CustomerName:       command.CustomerName,
		ReferenceID:        command.ReferenceID,
	})
	if err != nil {
		h.eventBus.Publish(ctx, entities.TaxiBookingFailed_v1{
			Header:        entities.NewEventHeader(),
			FailureReason: err.Error(),
			ReferenceID:   command.ReferenceID,
		})

		return fmt.Errorf("failed to book taxi: %w", err)
	}

	err = h.eventBus.Publish(ctx, entities.TaxiBooked_v1{
		Header:        entities.NewEventHeader(),
		TaxiBookingID: resp,
		ReferenceID:   command.ReferenceID,
	})

	if err != nil {
		return fmt.Errorf("failed to book taxi: %w", err)
	}

	return nil
}
