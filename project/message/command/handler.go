package command

import (
	"context"
	"tickets/entities"
)

type ReceiptsService interface {
	RefundVoidReceipts(ctx context.Context, cmd entities.TicketRefunded) error
	RefundPayment(ctx context.Context, cmd entities.TicketRefunded) error
}
type Handler struct {
	receiptsService ReceiptsService
}

func NewHandler(receiptsService ReceiptsService) Handler {
	if receiptsService == nil {
		panic("missin spreedsheetsService")
	}

	return Handler{
		receiptsService: receiptsService,
	}
}
