package command

import (
	"context"
	"tickets/entities"
)

func (h Handler) CancelFlightTickets(ctx context.Context, command *entities.CancelFlightTickets) error {
	for _, ticketID := range command.FlightTicketIDs {
		err := h.transportaionService.DeleteFlightTickets(ctx, ticketID)
		if err != nil {
			panic(err)
		}
	}
	return nil
}
