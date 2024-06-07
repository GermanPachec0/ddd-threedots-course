package entities

type TicketRefunded struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}
