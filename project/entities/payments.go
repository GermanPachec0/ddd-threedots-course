package entities

type PaymentRefund struct {
	TicketID       string
	RefundReason   string
	IdempotencyKey string
}
