package entities

type TicketRefunded_v1 struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}
