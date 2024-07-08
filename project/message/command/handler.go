package command

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
)

type ReceiptsService interface {
	VoidReceipt(ctx context.Context, request entities.VoidReceipt) error
}

type PaymentsService interface {
	RefundPayment(ctx context.Context, request entities.PaymentRefund) error
}

type TransportationService interface {
	BookFlight(ctx context.Context, bookFlight entities.BookFlightTicketRequest) (entities.BookFlightTicketResponse, error)
	BookTaxi(ctx context.Context, bookTaxi entities.BookTaxi) (uuid.UUID, error)
	DeleteFlightTickets(ctx context.Context, ticketID uuid.UUID) error
}

type Handler struct {
	receiptsService       ReceiptsService
	bookingsRepo          BookingsRepository
	transportaionService  TransportationService
	eventBus              *cqrs.EventBus
	commandBus            *cqrs.CommandBus
	paymentsServiceClient PaymentsService
}
type BookingsRepository interface {
	Create(ctx context.Context, booking entities.Booking) (entities.BookingCreateResponse, error)
}

func NewHandler(eventBus *cqrs.EventBus,
	receiptsServiceClient ReceiptsService,
	bookingsRepo BookingsRepository,
	transportaionService TransportationService,
	commandBus *cqrs.CommandBus,
	paymentsService PaymentsService) Handler {
	if eventBus == nil {
		panic("eventBus is required")
	}
	if receiptsServiceClient == nil {
		panic("receiptsServiceClient is required")
	}

	handler := Handler{
		eventBus:              eventBus,
		receiptsService:       receiptsServiceClient,
		bookingsRepo:          bookingsRepo,
		transportaionService:  transportaionService,
		commandBus:            commandBus,
		paymentsServiceClient: paymentsService,
	}

	return handler
}
