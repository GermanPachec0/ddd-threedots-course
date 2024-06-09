package http

import (
	"net/http"

	libHttp "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/labstack/echo/v4"
)

func NewHttpRouter(
	eventBus *cqrs.EventBus,
	cmdBus *cqrs.CommandBus,
	spreadsheetsAPIClient SpreadsheetsAPI,
	ticketRepo TicketRepository,
	showRepo ShowRepository,
	bookingRepo BookingRespository,
	opsBookingRepo OpsBookingRepository,
) *echo.Echo {
	e := libHttp.NewEcho()

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	handler := Handler{
		eventBus:              eventBus,
		cmdBus:                cmdBus,
		spreadsheetsAPIClient: spreadsheetsAPIClient,
		ticketRepo:            ticketRepo,
		showRepo:              showRepo,
		bookingRepo:           bookingRepo,
		opsBookingRepo:        opsBookingRepo,
	}

	e.POST("/tickets-status", handler.PostTicketsStatus)
	e.POST("/book-tickets", handler.PostBookTickets)
	e.PUT("/ticket-refund/:ticket_id", handler.PutTicketRefund)
	e.POST("/shows", handler.PostShows)
	e.GET("/tickets", handler.GetTickets)
	e.GET("/ops/bookings", handler.GetBookings)
	e.GET("/ops/bookings/:id", handler.GetBookingsByID)

	return e
}
