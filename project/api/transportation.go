package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/transportation"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

type Transportation struct {
	clients *clients.Clients
}

type TicketResponse struct {
	TicketsID uuid.UUIDs `json:"ticket_ids"`
}

func NewTransportationClient(clients *clients.Clients) *Transportation {
	if clients == nil {
		panic("client is null")
	}

	return &Transportation{clients: clients}
}

func (t Transportation) BookFlight(
	ctx context.Context,
	request entities.BookFlightTicketRequest,
) (entities.BookFlightTicketResponse, error) {
	resp, err := t.clients.Transportation.PutFlightTicketsWithResponse(ctx, transportation.BookFlightTicketRequest{
		CustomerEmail:  request.CustomerEmail,
		FlightId:       request.FlightID,
		PassengerNames: request.PassengerNames,
		ReferenceId:    request.ReferenceId,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		return entities.BookFlightTicketResponse{}, fmt.Errorf("failed to book flight: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusCreated:
		return entities.BookFlightTicketResponse{
			TicketIds: lo.Map(resp.JSON201.TicketIds, func(i openapi_types.UUID, _ int) uuid.UUID {
				return i
			}),
		}, nil
	case http.StatusConflict:
		return entities.BookFlightTicketResponse{}, entities.ErrNoFlightTicketsAvailable
	default:
		return entities.BookFlightTicketResponse{}, fmt.Errorf(
			"unexpected status code for PUT transportation-api/transportation/flight-tickets: %d",
			resp.StatusCode(),
		)
	}
}

func (t Transportation) DeleteFlightTickets(ctx context.Context, ticketID uuid.UUID) error {
	_, err := t.clients.Transportation.DeleteFlightTicketsTicketIdWithResponse(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to cancel flight tickets: %w", err)
	}

	return nil
}

func (t Transportation) BookTaxi(ctx context.Context, bookTaxi entities.BookTaxi) (uuid.UUID, error) {
	resp, err := t.clients.Transportation.PutTaxiBookingWithResponse(ctx, transportation.TaxiBookingRequest{
		CustomerEmail:      bookTaxi.CustomerEmail,
		NumberOfPassengers: bookTaxi.NumberOfPassengers,
		PassengerName:      bookTaxi.CustomerName,
		ReferenceId:        bookTaxi.ReferenceID,
		IdempotencyKey:     bookTaxi.IdempotencyKey,
	})

	if err != nil {
		return uuid.Nil, err
	}

	if resp.StatusCode() == http.StatusBadRequest {
		return uuid.Nil, fmt.Errorf("bad request %w", resp.Body)
	}
	var bookingID uuid.UUID
	err = json.Unmarshal(resp.Body, &bookingID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error marshaling booking taxi: %w", err)
	}
	return bookingID, nil
}
