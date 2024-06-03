package http

import (
	"net/http"
	"tickets/entities"

	"github.com/labstack/echo/v4"
)

func (h *Handler) PostShows(c echo.Context) error {
	var showRequest entities.Show

	err := c.Bind(&showRequest)
	if err != nil {
		return err
	}

	ticketResponse, err := h.showRepo.Create(c.Request().Context(), entities.Show{
		DeadNationID:    showRequest.DeadNationID,
		NumberOfTickets: showRequest.NumberOfTickets,
		StartTime:       showRequest.StartTime,
		Title:           showRequest.Title,
		Venue:           showRequest.Venue,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, ticketResponse)
}
