package http

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
)

type Handler struct {
	eventBus              *cqrs.EventBus
	cmdBus                *cqrs.CommandBus
	spreadsheetsAPIClient SpreadsheetsAPI
	ticketRepo            TicketRepository
	showRepo              ShowRepository
	bookingRepo           BookingRespository
}

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, spreadsheetName string, row []string) error
}

type TicketRepository interface {
	Get(ctx context.Context) ([]entities.Ticket, error)
}

type ShowRepository interface {
	Create(ctx context.Context, show entities.Show) (entities.ShowCreateResponse, error)
	ShowByID(ctx context.Context, showID uuid.UUID) (entities.Show, error)
}

type BookingRespository interface {
	Create(ctx context.Context, booking entities.Booking) (entities.BookingCreateResponse, error)
}
