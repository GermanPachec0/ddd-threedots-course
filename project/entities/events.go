package entities

import (
	"time"

	"github.com/google/uuid"
)

type EventHeader struct {
	ID             string    `json:"id"`
	PublishedAt    time.Time `json:"published_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		ID:             uuid.NewString(),
		PublishedAt:    time.Now().UTC(),
		IdempotencyKey: uuid.NewString(),
	}
}

func NewEventHeaderWithIdempotencyKey(idempotencyKey string) EventHeader {
	return EventHeader{
		ID:             uuid.NewString(),
		PublishedAt:    time.Now().UTC(),
		IdempotencyKey: idempotencyKey,
	}
}

type TicketBookingConfirmed struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id" db:"ticket_id"`
	CustomerEmail string `json:"customer_email" db:"customer_email"`
	Price         Money  `json:"price" db:"price"`

	BookingID string `json:"booking_id"`
}

type TicketBookingCanceled struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id" db:"ticket_id"`
	CustomerEmail string `json:"customer_email" customer_email"`
	Price         Money  `json:"price" db:"price"`
}

type BookingMade struct {
	Header EventHeader `json:"header"`

	NumberOfTickets int `json:"number_of_tickets"`

	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail string    `json:"customer_email"`
	ShowId        uuid.UUID `json:"show_id"`
}
type TicketPrinted struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
	FileName string `json:"file_name"`
}

type TicketReceiptIssued struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id"`
	ReceiptNumber string `json:"receipt_number"`

	IssuedAt time.Time `json:"issued_at"`
}

type OpsBooking struct {
	BookingID uuid.UUID `json:"booking_id"`
	BookedAt  time.Time `json:"booked_at"`

	Tickets map[string]OpsTicket `json:"tickets"`

	LastUpdate time.Time `json:"last_update"`
}

type OpsTicket struct {
	PriceAmount   string `json:"price_amount"`
	PriceCurrency string `json:"price_currency"`
	CustomerEmail string `json:"customer_email"`

	// Status should be set to "confirmed" or "refunded"
	Status string `json:"status"`

	PrintedAt       time.Time `json:"printed_at"`
	PrintedFileName string    `json:"printed_file_name"`

	ReceiptIssuedAt time.Time `json:"receipt_issued_at"`
	ReceiptNumber   string    `json:"receipt_number"`
}
