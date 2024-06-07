package command

import (
	"context"
	"tickets/entities"
)

func (h *Handler) RefundTicket(ctx context.Context, cmd *entities.TicketRefunded) error {
	err := h.receiptsService.RefundPayment(ctx, entities.TicketRefunded{
		Header:   cmd.Header,
		TicketID: cmd.TicketID,
	})
	if err != nil {
		return err
	}
	err = h.receiptsService.RefundVoidReceipts(ctx, entities.TicketRefunded{
		Header:   cmd.Header,
		TicketID: cmd.TicketID,
	})
	if err != nil {
		return err
	}
	return nil
}
