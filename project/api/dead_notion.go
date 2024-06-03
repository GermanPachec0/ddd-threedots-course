package api

import (
	"context"
	"fmt"
	"net/http"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/dead_nation"
)

type DeadNotionClient struct {
	clients *clients.Clients
}

func NewDeadNotionClient(clients *clients.Clients) *DeadNotionClient {
	if clients == nil {
		panic("New Spread sheets clients is nil")
	}

	return &DeadNotionClient{clients: clients}
}

func (dn DeadNotionClient) CreateBooking(ctx context.Context, booking entities.DeadNationBookingRequest) error {
	resp, err := dn.clients.DeadNation.PostTicketBookingWithResponse(
		ctx,
		dead_nation.PostTicketBookingRequest{
			BookingId:       booking.BookingID,
			CustomerAddress: booking.CustomerEmail,
			EventId:         booking.DeadNationEventID,
			NumberOfTickets: booking.NumberOfTickets,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to book place in Dead Nation: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code from dead nation: %d", resp.StatusCode())
	}

	return nil
}
