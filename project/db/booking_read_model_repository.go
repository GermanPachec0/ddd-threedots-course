package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"tickets/entities"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/jmoiron/sqlx"
)

type OpsBookingReadModel struct {
	conn *DB
}

func NewOpsBookingReadModel(db *DB) OpsBookingReadModel {
	return OpsBookingReadModel{
		conn: db,
	}
}

func (r OpsBookingReadModel) OnBookingMade(ctx context.Context, bookingMade *entities.BookingMade) error {
	// this is the first event that should arrive, so we create the read model
	err := r.createReadModel(ctx, entities.OpsBooking{
		BookingID:  bookingMade.BookingID,
		Tickets:    nil,
		LastUpdate: time.Now(),
		BookedAt:   bookingMade.Header.PublishedAt,
	})
	if err != nil {
		return fmt.Errorf("could not create read model: %w", err)
	}

	return nil
}
func (r OpsBookingReadModel) OnTicketBookingConfirmed(ctx context.Context, event *entities.TicketBookingConfirmed) error {
	return r.updateBookingReadModel(
		ctx,
		event.BookingID,
		func(rm entities.OpsBooking) (entities.OpsBooking, error) {

			ticket, ok := rm.Tickets[event.TicketID]
			if !ok {
				// we are using zero-value of OpsTicket
				log.
					FromContext(ctx).
					WithField("ticket_id", event.TicketID).
					Debug("Creating ticket read model for ticket %s")
			}

			ticket.PriceAmount = event.Price.Amount
			ticket.PriceCurrency = event.Price.Currency
			ticket.CustomerEmail = event.CustomerEmail
			ticket.Status = "confirmed"

			rm.Tickets[event.TicketID] = ticket

			return rm, nil
		},
	)
}
func (r OpsBookingReadModel) OnTicketReceiptIssued(ctx context.Context, issued *entities.TicketReceiptIssued) error {
	return r.updateTicketReadModel(
		ctx,
		issued.TicketID,
		func(rm entities.OpsTicket) (entities.OpsTicket, error) {
			rm.ReceiptIssuedAt = issued.IssuedAt
			rm.ReceiptNumber = issued.ReceiptNumber

			return rm, nil
		},
	)
}

func (r OpsBookingReadModel) OnTicketPrinted(ctx context.Context, event *entities.TicketPrinted) error {
	return r.updateTicketReadModel(
		ctx,
		event.TicketID,
		func(rm entities.OpsTicket) (entities.OpsTicket, error) {
			rm.PrintedAt = time.Now()
			rm.PrintedFileName = event.FileName
			return rm, nil
		},
	)
}

func (r OpsBookingReadModel) OnTicketRefunded(ctx context.Context, event *entities.TicketRefunded) error {
	return r.updateTicketReadModel(
		ctx,
		event.TicketID,
		func(rm entities.OpsTicket) (entities.OpsTicket, error) {
			rm.Status = "refunded"

			return rm, nil
		},
	)
}
func (r OpsBookingReadModel) createReadModel(ctx context.Context, opsBooking entities.OpsBooking) error {
	payload, err := json.Marshal(opsBooking)
	if err != nil {
		return err
	}

	_, err = r.conn.Conn.ExecContext(ctx, `
		INSERT INTO 
		    read_model_ops_bookings (payload, booking_id)
		VALUES
			($1, $2)
		ON CONFLICT (booking_id) DO NOTHING; -- read model may be already updated by another event - we don't want to override
`, payload, opsBooking.BookingID)

	if err != nil {
		return fmt.Errorf("could not create read model: %w", err)
	}

	return nil
}

func (r OpsBookingReadModel) updateTicketReadModel(ctx context.Context,
	ticketID string,
	updateFunc func(ticket entities.OpsTicket) (entities.OpsTicket, error),
) error {
	return updateInTx(
		ctx,
		r.conn.Conn,
		sql.LevelRepeatableRead,
		func(ctx context.Context, tx *sqlx.Tx) error {
			rm, err := r.findModelByTicketID(ctx, ticketID, tx)
			if err == sql.ErrNoRows {
				// events arrived out of order - it should spin until the read model is created
				return fmt.Errorf("read model for ticket %s not exist yet", ticketID)
			} else if err != nil {
				return fmt.Errorf("could not find read model: %w", err)
			}
			ticket, ok := rm.Tickets[ticketID]
			if !ok {
				return fmt.Errorf("ticket not found with id: %s", ticketID)
			}

			updatedRm, err := updateFunc(ticket)
			if err != nil {
				return err
			}

			rm.Tickets[ticketID] = updatedRm

			return r.updateModel(ctx, tx, rm)
		},
	)
}

func (r OpsBookingReadModel) updateBookingReadModel(ctx context.Context,
	bookingID string,
	updateFunc func(ticket entities.OpsBooking) (entities.OpsBooking, error),
) error {
	return updateInTx(ctx,
		r.conn.Conn,
		sql.LevelRepeatableRead,
		func(ctx context.Context, tx *sqlx.Tx) error {
			rm, err := r.findModelByBookingID(ctx, bookingID, tx)
			if err == sql.ErrNoRows {
				// events arrived out of order - it should spin until the read model is created
				return fmt.Errorf("read model for booking %s not exist yet", bookingID)
			} else if err != nil {
				return fmt.Errorf("could not find read model: %w", err)
			}

			updatedRm, err := updateFunc(rm)
			if err != nil {
				return err
			}

			return r.updateModel(ctx, tx, updatedRm)
		},
	)
}

func (r OpsBookingReadModel) updateModel(ctx context.Context, tx *sqlx.Tx, readModel entities.OpsBooking) error {
	readModel.LastUpdate = time.Now()

	payload, err := json.Marshal(readModel)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO 
			read_model_ops_bookings (payload, booking_id)
		VALUES
			($1, $2)
		ON CONFLICT (booking_id) DO UPDATE SET payload = excluded.payload;
		`, payload, readModel.BookingID)
	if err != nil {
		return fmt.Errorf("could not update read model: %w", err)
	}

	return nil
}

func (r OpsBookingReadModel) findModelByBookingID(ctx context.Context, bookingID string, tx *sqlx.Tx) (entities.OpsBooking, error) {
	var payload []byte

	err := r.conn.Conn.QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE booking_id = $1",
		bookingID,
	).Scan(&payload)
	if err != nil {
		return entities.OpsBooking{}, err
	}

	return r.unmarshalReadModelFromDB(payload)

}

func (r OpsBookingReadModel) findModelByTicketID(ctx context.Context, ticketID string, tx *sqlx.Tx) (entities.OpsBooking, error) {
	var payload []byte

	err := r.conn.Conn.QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE payload::jsonb -> 'tickets' ? $1",
		ticketID,
	).Scan(&payload)
	if err != nil {
		return entities.OpsBooking{}, err
	}

	return r.unmarshalReadModelFromDB(payload)

}

func (r OpsBookingReadModel) unmarshalReadModelFromDB(payload []byte) (entities.OpsBooking, error) {
	var opsReadModel entities.OpsBooking

	err := json.Unmarshal(payload, &opsReadModel)
	if err != nil {
		return entities.OpsBooking{}, err
	}

	if opsReadModel.Tickets == nil {
		opsReadModel.Tickets = map[string]entities.OpsTicket{}
	}
	return opsReadModel, nil
}

func updateInTx(
	ctx context.Context,
	db *sqlx.DB,
	isolation sql.IsolationLevel,
	fn func(ctx context.Context, tx *sqlx.Tx) error,
) (err error) {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: isolation})
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
			return
		}

		err = tx.Commit()
	}()

	return fn(ctx, tx)
}
