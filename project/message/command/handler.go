package command

import (
	"context"
	"tickets/entities"
)

type ReceiptsService interface {
	RefundVoidReceipts(ctx context.Context, cmd entities.RefundTicket) error
	RefundPayment(ctx context.Context, cmd entities.RefundTicket) error
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
