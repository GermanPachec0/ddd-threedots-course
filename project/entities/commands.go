package entities

import (
	"fmt"

	"github.com/google/uuid"
)

var ErrNoFlightTicketsAvailable = fmt.Errorf("no flight tickets available")

type TicketRefunded_v1 struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}

type RefundTicket struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}

type BookShowTickets struct {
	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail   string    `json:"customer_email"`
	NumberOfTickets int       `json:"number_of_tickets"`
	ShowId          uuid.UUID `json:"show_id"`
}

type BookFlight struct {
	CustomerEmail  string    `json:"customer_email"`
	FlightID       uuid.UUID `json:"to_flight_id"`
	Passengers     []string  `json:"passengers"`
	ReferenceID    string    `json:"reference_id"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type BookTaxi struct {
	CustomerEmail      string `json:"customer_email"`
	CustomerName       string `json:"customer_name"`
	NumberOfPassengers int    `json:"number_of_passengers"`
	ReferenceID        string `json:"reference_id"`
	IdempotencyKey     string `json:"idempotency_key"`
}

type CancelFlightTickets struct {
	FlightTicketIDs []uuid.UUID `json:"flight_ticket_id"`
}
