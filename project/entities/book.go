package entities

import "github.com/google/uuid"

type Booking struct {
	BookingID       uuid.UUID `json:"booking_id" db:"booking_id"`
	ShowID          uuid.UUID `json:"show_id" db:"show_id"`
	NumberOfTickets int       `json:"number_of_tickets" db:"number_of_tickets"`
	CustomerEmail   string    `json:"customer_email" db:"customer_email"`
}

type BookingCreateResponse struct {
	BookingID uuid.UUID `json:"booking_id"`
}

type DeadNationBookingRequest struct {
	BookingID         uuid.UUID
	NumberOfTickets   int
	CustomerEmail     string
	DeadNationEventID uuid.UUID
}
