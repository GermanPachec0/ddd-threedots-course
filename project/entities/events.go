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

type TicketBookingConfirmed_v1 struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id" db:"ticket_id"`
	CustomerEmail string `json:"customer_email" db:"customer_email"`
	Price         Money  `json:"price" db:"price"`

	BookingID string `json:"booking_id"`
}

type TicketBookingCanceled_v1 struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id" db:"ticket_id"`
	CustomerEmail string `json:"customer_email" db:"customer_email"`
	Price         Money  `json:"price" db:"price"`
}

type BookingMade_v1 struct {
	Header EventHeader `json:"header"`

	NumberOfTickets int `json:"number_of_tickets"`

	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail string    `json:"customer_email"`
	ShowId        uuid.UUID `json:"show_id"`
}
type TicketPrinted_v1 struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
	FileName string `json:"file_name"`
}

type TicketReceiptIssued_v1 struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id"`
	ReceiptNumber string `json:"receipt_number"`

	IssuedAt time.Time `json:"issued_at"`
}

type OpsBooking_v1 struct {
	BookingID uuid.UUID `json:"booking_id" db:"booking_id"`
	BookedAt  time.Time `json:"booked_at" db:"booked_at" `

	Tickets map[string]OpsTicket_v1 `json:"tickets" db:"tickets"`

	LastUpdate time.Time `json:"last_update" db:"last_update"`
}

type OpsTicket_v1 struct {
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
	EventPayload []byte      `json:"event_payload" db:"event_payload"`
}

type IEvent interface {
	IsInternal() bool
}

func (i Event) IsInternal() bool {
	return false
}

func (i TicketBookingCanceled_v1) IsInternal() bool {
	return false
}

func (i TicketBookingConfirmed_v1) IsInternal() bool {
	return false
}

func (i TicketPrinted_v1) IsInternal() bool {
	return false
}
func (i TicketReceiptIssued_v1) IsInternal() bool {
	return false
}

func (i TicketRefunded_v1) IsInternal() bool {
	return false
}
func (i OpsTicket_v1) IsInternal() bool {
	return false
}

func (i BookingMade_v1) IsInternal() bool {
	return false
}
