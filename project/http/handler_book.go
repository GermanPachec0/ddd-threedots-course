package http

import (
	"net/http"
	"tickets/entities"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) PostBookTickets(c echo.Context) error {
	var bookReq entities.Booking

	err := c.Bind(&bookReq)
	if err != nil {
		return err
	}

	if bookReq.NumberOfTickets < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "number of tickets must be greater than 0")
	}

	bookReq.BookingID = uuid.New()

	bookResp, err := h.bookingRepo.Create(c.Request().Context(), bookReq)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusCreated, bookResp)
}
