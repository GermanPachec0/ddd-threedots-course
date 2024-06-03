package entities

import "time"

type VoidReceipt struct {
	TicketID       string
	Reason         string
	IdempotencyKey string
}

type IssueReceiptRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	TicketID       string `json:"ticket_id"`
	Price          Money  `json:"price"`
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}
