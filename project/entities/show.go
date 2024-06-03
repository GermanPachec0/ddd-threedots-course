package entities

import (
	"time"

	"github.com/google/uuid"
)

type Show struct {
	ShowID          uuid.UUID `json:"show_id" db:"show_id"`
	DeadNationID    uuid.UUID `json:"dead_nation_id" db:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets" db:"number_of_tickets"`
	StartTime       time.Time `json:"start_time" db:"start_time"`
	Title           string    `json:"title" db:"title"`
	Venue           string    `json:"venue" db:"venue"`
}

type ShowCreateResponse struct {
	ShowID uuid.UUID `json:"show_id"`
}
