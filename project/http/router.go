package http

import (
	"net/http"

	libHttp "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func NewHttpRouter(
	eventBus *cqrs.EventBus,
	cmdBus *cqrs.CommandBus,
	spreadsheetsAPIClient SpreadsheetsAPI,
	ticketRepo TicketRepository,
	showRepo ShowRepository,
	bookingRepo BookingRespository,
	opsBookingRepo OpsBookingRepository,
	vipBundleRepo VipBundleRepository,
) *echo.Echo {
	e := libHttp.NewEcho()
	e.Use(otelecho.Middleware("tickets"))

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	handler := Handler{
		eventBus:              eventBus,
		cmdBus:                cmdBus,
		spreadsheetsAPIClient: spreadsheetsAPIClient,
		ticketRepo:            ticketRepo,
		showRepo:              showRepo,
		bookingRepo:           bookingRepo,
		opsBookingRepo:        opsBookingRepo,
		vipBundleRepo:         vipBundleRepo,
	}

	e.POST("/tickets-status", handler.PostTicketsStatus)
	e.POST("/book-vip-bundle", handler.PostVipBundler)
	e.POST("/book-tickets", handler.PostBookTickets)
	e.PUT("/ticket-refund/:ticket_id", handler.PutTicketRefund)
	e.POST("/shows", handler.PostShows)
	e.GET("/tickets", handler.GetTickets)
	e.GET("/ops/bookings", handler.GetBookings)
	e.GET("/ops/bookings/:id", handler.GetBookingsByID)

	return e
}
