package entities

import "time"

type Ticket struct {
	TicketID      string     `json:"ticket_id" db:"ticket_id"`
	Price         Money      `json:"price" db:"price"`
	CustomerEmail string     `json:"customer_email" db:"customer_email"`
	DeleteAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
