package http

import (
	"fmt"
	"net/http"
	"tickets/entities"

	"github.com/labstack/echo/v4"
)

type ticketsStatusRequest struct {
	Tickets []ticketStatusRequest `json:"tickets"`
}

type ticketStatusRequest struct {
	TicketID      string         `json:"ticket_id"`
	Status        string         `json:"status"`
	Price         entities.Money `json:"price"`
	CustomerEmail string         `json:"customer_email"`
	BookingID     string         `json:"booking_id"`
}

func (h Handler) GetTickets(c echo.Context) error {
	tickets, err := h.ticketRepo.Get(c.Request().Context())
	if err != nil {
		return fmt.Errorf("failed getting tickets %w", err)
	}

	return c.JSON(http.StatusOK, tickets)
}

func (h Handler) PostTicketsStatus(c echo.Context) error {
	var request ticketsStatusRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Idempotency-Key header is required")
	}
	for _, ticket := range request.Tickets {
		if ticket.Status == "confirmed" {
			event := entities.TicketBookingConfirmed{
				Header: entities.NewEventHeaderWithIdempotencyKey(idempotencyKey + ticket.TicketID),

				TicketID:      ticket.TicketID,
				Price:         ticket.Price,
				CustomerEmail: ticket.CustomerEmail,

				BookingID: ticket.BookingID,
			}

			if err := h.eventBus.Publish(c.Request().Context(), event); err != nil {
				return fmt.Errorf("failed to publish TicketBookingConfirmed event: %w", err)
			}
		} else if ticket.Status == "canceled" {
			event := entities.TicketBookingCanceled{
				Header:        entities.NewEventHeaderWithIdempotencyKey(idempotencyKey + ticket.TicketID),
				TicketID:      ticket.TicketID,
				CustomerEmail: ticket.CustomerEmail,
				Price:         ticket.Price,
			}

			if err := h.eventBus.Publish(c.Request().Context(), event); err != nil {
				return fmt.Errorf("failed to publish TicketBookingCanceled event: %w", err)
			}
		} else {
			return fmt.Errorf("unknown ticket status: %s", ticket.Status)
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) PutTicketRefund(c echo.Context) error {
	ticketId := c.Param("ticket_id")

	if ticketId == "" {
		return fmt.Errorf("ticket id not provided")
	}
	cmd := entities.RefundTicket{
		Header:   entities.NewEventHeaderWithIdempotencyKey(ticketId),
		TicketID: ticketId,
	}

	if err := h.cmdBus.Send(c.Request().Context(), cmd); err != nil {
		return fmt.Errorf("failed to send ticket refunded command: %w", err)
	}
	return c.NoContent(http.StatusAccepted)
}
