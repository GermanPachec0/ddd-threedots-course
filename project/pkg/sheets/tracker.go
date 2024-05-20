package sheets

import "tickets/adapters/echo_server/ticket"

type AppendToTrackerPayload struct {
	TicketID      string       `json:"ticket_id"`
	CustomerEmail string       `json:"customer_email"`
	Price         ticket.Price `json:"price"`
}
