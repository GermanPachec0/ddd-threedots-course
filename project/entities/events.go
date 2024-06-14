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
	CustomerEmail string `json:"customer_email" db:"customer_email"`
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
	BookingID uuid.UUID `json:"booking_id" db:"booking_id"`
	BookedAt  time.Time `json:"booked_at" db:"booked_at" `

	Tickets map[string]OpsTicket `json:"tickets" db:"tickets"`

	LastUpdate time.Time `json:"last_update" db:"last_update"`
}

type OpsTicket struct {
	PriceAmount   string `json:"price_amount"`
	PriceCurrency string `json:"price_currency"`
	CustomerEmail string `json:"customer_email"`

	// Status should be set to "confirmed" or "refunded"
	ConfirmedAt time.Time `json:"confirmed_at"`
	RefundedAt  time.Time `json:"refunded_at"`

	PrintedAt       time.Time `json:"printed_at"`
	PrintedFileName string    `json:"printed_file_name"`

	ReceiptIssuedAt time.Time `json:"receipt_issued_at"`
	ReceiptNumber   string    `json:"receipt_number"`
}

type Event struct {
	Header       EventHeader `json:"header"`
	EventID      string      `json:"event_id" db:"event_id"`
	PublishedAt  time.Time   `json:"published_at" db:"published_at"`
	EventName    string      `json:"event_name" db:"event_name"`
	EventPayload string      `json:"event_payload" db:"event_payload"`
}
