package ticket

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	publisher message.Publisher
}

func NewController(pub message.Publisher) *Controller {
	return &Controller{publisher: pub}
}

func (tc *Controller) ProcessTicket(c echo.Context) error {
	var request TicketsConfirmationRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}
	corrId := c.Request().Header.Get("Correlation-ID")
	for _, ticket := range request.Tickets {
		if ticket.Status == "canceled" {
			err = tc.PublishBookingCanceled(ticket, corrId)
			if err != nil {
				return err
			}
		} else if ticket.Status == "confirmed" {
			err = tc.PublishBookingConfirmed(ticket, corrId)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown ticket status: %s", ticket.Status)
		}

	}

	return c.NoContent(http.StatusOK)
}

func (tc *Controller) PublishBookingConfirmed(payload Ticket, correlationID string) error {
	h := NewHeader()
	payload.Header = h

	msg, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msgSend := message.NewMessage(watermill.NewUUID(), msg)
	msgSend.Metadata.Set("correlation_id", correlationID)
	msgSend.Metadata.Set("type", "TicketBookingConfirmed")

	return tc.publisher.Publish("TicketBookingConfirmed", msgSend)
}

func (tc *Controller) PublishBookingCanceled(payload Ticket, correlationID string) error {
	h := NewHeader()
	payload.Header = h

	msg, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msgSend := message.NewMessage(watermill.NewUUID(), msg)
	msgSend.Metadata.Set("correlation_id", correlationID)
	msgSend.Metadata.Set("type", "TicketBookingCanceled")

	return tc.publisher.Publish("TicketBookingCanceled", msgSend)
}

func (tc *Controller) Health(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}
