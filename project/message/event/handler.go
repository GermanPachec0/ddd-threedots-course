package event

import (
	"context"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
)

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, sheetName string, row []string) error
}

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request entities.IssueReceiptRequest) (entities.IssueReceiptResponse, error)
	RefundVoidReceipts(ctx context.Context, cmd entities.TicketRefunded) error
	RefundPayment(ctx context.Context, cmd entities.TicketRefunded) error
}
type ShowRepository interface {
	ShowByID(ctx context.Context, showID uuid.UUID) (entities.Show, error)
}
type TicketRepository interface {
	Create(ctx context.Context, ticket entities.Ticket) error
	Delete(ctx context.Context, ticket entities.Ticket) error
}

type FileService interface {
	StoreFile(ctx context.Context, ticketFile string, ticketHTML string) error
}

type DeadNationService interface {
	CreateBooking(ctx context.Context, booking entities.DeadNationBookingRequest) error
}

type Handler struct {
	spreadsheetsService SpreadsheetsAPI
	receiptsService     ReceiptsService
	fileService         FileService
	eventBus            *cqrs.EventBus
	ticketRepo          TicketRepository
	showRepo            ShowRepository
	deadNationSvc       DeadNationService
}

func NewHandler(spreedsheetsService SpreadsheetsAPI, receiptsService ReceiptsService, ticketRepo TicketRepository, fileService FileService,
	eventBus *cqrs.EventBus, deadNationService DeadNationService, showRepo ShowRepository) Handler {
	if spreedsheetsService == nil {
		panic("missin spreedsheetsService")
	}
	if spreedsheetsService == nil {
		panic("missing receiptsService")
	}
	return Handler{
		spreadsheetsService: spreedsheetsService,
		receiptsService:     receiptsService,
		ticketRepo:          ticketRepo,
		fileService:         fileService,
		eventBus:            eventBus,
		deadNationSvc:       deadNationService,
		showRepo:            showRepo,
	}
}
