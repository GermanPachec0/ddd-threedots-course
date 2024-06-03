package entities

type RefundTicket struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}
