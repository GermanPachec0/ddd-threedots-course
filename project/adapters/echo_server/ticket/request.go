package ticket

import (
	"time"

	"github.com/google/uuid"
)

type Header struct {
	ID          string `json:"id"`
	PublishedAt string `json:"published_at"`
}
type Ticket struct {
	Header        Header `json:"header"`
	ID            string `json:"ticket_id"`
	Status        string `json:"status"`
	CustomerEmail string `json:"customer_email"`
	Price         Price  `json:"price"`
}

type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type TicketsConfirmationRequest struct {
	Tickets []Ticket `json:"tickets"`
}

func NewHeader() Header {
	return Header{
		ID:          uuid.NewString(),
		PublishedAt: time.Now().Format(time.RFC3339),
	}
}
