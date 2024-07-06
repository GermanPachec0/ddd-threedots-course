package http

import (
	"context"
	"tickets/entities"
	"tickets/message/sagas"

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
	opsBookingRepo        OpsBookingRepository
	vipBundleRepo         VipBundleRepository
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

type VipBundleRepository interface {
	Add(ctx context.Context, vipBundle sagas.VipBundle) error
}

type OpsBookingRepository interface {
	GetAll(ctx context.Context, query *string) ([]entities.OpsBooking_v1, error)
	GetByID(ctx context.Context, bookingID string) (entities.OpsBooking_v1, error)
}
