package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"tickets/entities"
	"tickets/message/event"
	"tickets/message/outbox"

	"github.com/labstack/echo/v4"
)

type IBookingRepository interface {
	Create(ctx context.Context, booking entities.Booking) (entities.BookingCreateResponse, error)
}

type BookingRepository struct {
	db       *DB
	showRepo ShowRepository
}

func NewBookingRespository(db *DB) BookingRepository {
	if db == nil {
		panic("db is nil")
	}
	return BookingRepository{
		db:       db,
		showRepo: NewShowRepository(db),
	}
}

func (br BookingRepository) Create(ctx context.Context, booking entities.Booking) (entities.BookingCreateResponse, error) {
	tx, err := br.db.Conn.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable})
	if err != nil {
		return entities.BookingCreateResponse{}, fmt.Errorf("could not begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			err = errors.Join(err, rollbackErr)
			return
		}
		err = tx.Commit()
	}()

	availableSeats := 0
	err = tx.GetContext(ctx, &availableSeats, `
		SELECT
		    number_of_tickets AS available_seats
		FROM
		    shows
		WHERE
		    show_id = $1
	`, booking.ShowID)
	if err != nil {
		return entities.BookingCreateResponse{}, fmt.Errorf("could not get available seats: %w", err)
	}

	alreadyBookedSeats := 0
	err = tx.GetContext(ctx, &alreadyBookedSeats, `
		SELECT
		    coalesce(SUM(number_of_tickets), 0) AS already_booked_seats
		FROM
		    bookings
		WHERE
		    show_id = $1
	`, booking.ShowID)
	if err != nil {
		return entities.BookingCreateResponse{}, fmt.Errorf("could not get already booked seats: %w", err)
	}

	if availableSeats-alreadyBookedSeats < booking.NumberOfTickets {
		return entities.BookingCreateResponse{}, echo.NewHTTPError(http.StatusBadRequest, "not enough seats available")
	}

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO 
		    bookings (booking_id, show_id, number_of_tickets, customer_email) 
		VALUES (:booking_id, :show_id, :number_of_tickets, :customer_email)
		`, booking)
	if err != nil {
		return entities.BookingCreateResponse{}, fmt.Errorf("could not add booking: %w", err)
	}

	outBoxPuslisher, err := outbox.NewPublisherForDb(ctx, tx)
	if err != nil {
		return entities.BookingCreateResponse{}, fmt.Errorf("error creating event outbox publisher %w", err)
	}
	err = event.NewBus(outBoxPuslisher).Publish(ctx, entities.BookingMade_v1{
		Header:          entities.NewEventHeader(),
		BookingID:       booking.BookingID,
		NumberOfTickets: booking.NumberOfTickets,
		CustomerEmail:   booking.CustomerEmail,
		ShowId:          booking.ShowID,
	})
	if err != nil {
		return entities.BookingCreateResponse{}, fmt.Errorf("could not publish event: %w", err)
	}

	fmt.Println("Booking ID: %s", booking.BookingID)
	return entities.BookingCreateResponse{BookingID: booking.BookingID}, nil

}
