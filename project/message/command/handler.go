package command

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type ReceiptsService interface {
	RefundVoidReceipts(ctx context.Context, cmd entities.TicketRefunded_v1) error
	RefundPayment(ctx context.Context, cmd entities.TicketRefunded_v1) error
}
type Handler struct {
	receiptsService ReceiptsService
	eventBus        *cqrs.EventBus
}

func NewHandler(eventBus *cqrs.EventBus, receiptsServiceClient ReceiptsService) Handler {
	if eventBus == nil {
		panic("eventBus is required")
	}
	if receiptsServiceClient == nil {
		panic("receiptsServiceClient is required")
	}

	handler := Handler{
		eventBus:        eventBus,
		receiptsService: receiptsServiceClient,
	}

	return handler
}
