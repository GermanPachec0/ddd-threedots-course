package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *Handler) GetBookings(c echo.Context) error {

	date := c.QueryParam("receipt_issue_date")
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid receipt_issue_date format, expected RFC3339 date: ", err.Error())
	}

	resp, err := h.opsBookingRepo.GetAll(c.Request().Context(), &date)
	if err != nil {
		return fmt.Errorf("failed getting ops bookings %w", err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetBookingsByID(c echo.Context) error {
	bID := c.Param("id")

	resp, err := h.opsBookingRepo.GetByID(c.Request().Context(), bID)
	if err != nil {
		return fmt.Errorf("failed getting ops bookings %w", err)
	}

	return c.JSON(http.StatusOK, resp)
}
