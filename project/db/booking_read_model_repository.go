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
	conn *sqlx.DB
}

func NewOpsBookingReadModel(db *DB) OpsBookingReadModel {
	return OpsBookingReadModel{
		conn: db.Conn,
	}
}

func (r OpsBookingReadModel) GetAll(ctx context.Context, date *string) ([]entities.OpsBooking, error) {
	query := "SELECT payload FROM read_model_ops_bookings"
	var quaryArgs []any

	if date != nil {
		query += fmt.Sprintf(`
			WHERE booking_id IN (
				SELECT booking_id FROM (
					SELECT booking_id, 
						DATE(jsonb_path_query(payload, '$.tickets.*.receipt_issued_at')::text) as receipt_issued_at 
					FROM 
						read_model_ops_bookings
				) bookings_within_date 
				WHERE receipt_issued_at = $1
			)
		`)
		quaryArgs = append(quaryArgs, date)
	}

	rows, err := r.conn.Query(query, quaryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entities.OpsBooking
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}

		var booking entities.OpsBooking
		if err := json.Unmarshal(payload, &booking); err != nil {
			return nil, err
		}

		result = append(result, booking)
	}
	return result, nil
}

func (r OpsBookingReadModel) GetByID(ctx context.Context, bookingID string) (entities.OpsBooking, error) {
	var bookingPayload []byte

	err := r.conn.QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE booking_id = $1",
		bookingID,
	).Scan(&bookingPayload)
	if err != nil {
		return entities.OpsBooking{}, err
	}

	var booking entities.OpsBooking

	err = json.Unmarshal(bookingPayload, &booking)
	if err != nil {
		return entities.OpsBooking{}, err
	}

	return booking, nil
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
	return r.updateTicketInBookingReadModel(
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
	return r.updateTicketInBookingReadModel(
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
	return r.updateTicketInBookingReadModel(
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

	_, err = r.conn.ExecContext(ctx, `
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

func (r OpsBookingReadModel) updateTicketInBookingReadModel(
	ctx context.Context,
	ticketID string,
	updateFunc func(ticket entities.OpsTicket) (entities.OpsTicket, error),
) (err error) {
	return updateInTx(
		ctx,
		r.conn,
		sql.LevelRepeatableRead,
		func(ctx context.Context, tx *sqlx.Tx) error {
			rm, err := r.findReadModelByTicketID(ctx, ticketID, tx)
			if err == sql.ErrNoRows {
				// events arrived out of order - it should spin until the read model is created
				return fmt.Errorf("read model for ticket %s not exist yet", ticketID)
			} else if err != nil {
				return fmt.Errorf("could not find read model: %w", err)
			}

			ticket, _ := rm.Tickets[ticketID]

			updatedRm, err := updateFunc(ticket)
			if err != nil {
				return err
			}

			rm.Tickets[ticketID] = updatedRm

			return r.updateReadModel(ctx, tx, rm)
		},
	)
}

func (r OpsBookingReadModel) updateBookingReadModel(ctx context.Context,
	bookingID string,
	updateFunc func(ticket entities.OpsBooking) (entities.OpsBooking, error),
) error {
	return updateInTx(ctx,
		r.conn,
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

			return r.updateReadModel(ctx, tx, updatedRm)
		},
	)
}

func (r OpsBookingReadModel) updateReadModel(
	ctx context.Context,
	tx *sqlx.Tx,
	rm entities.OpsBooking,
) error {
	rm.LastUpdate = time.Now()

	payload, err := json.Marshal(rm)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO 
			read_model_ops_bookings (payload, booking_id)
		VALUES
			($1, $2)
		ON CONFLICT (booking_id) DO UPDATE SET payload = excluded.payload;
		`, payload, rm.BookingID)
	if err != nil {
		return fmt.Errorf("could not update read model: %w", err)
	}

	return nil
}

func (r OpsBookingReadModel) findModelByBookingID(ctx context.Context, bookingID string, tx *sqlx.Tx) (entities.OpsBooking, error) {
	var payload []byte

	err := r.conn.QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE booking_id = $1",
		bookingID,
	).Scan(&payload)
	if err != nil {
		return entities.OpsBooking{}, err
	}

	return r.unmarshalReadModelFromDB(payload)

}

func (r OpsBookingReadModel) findReadModelByTicketID(
	ctx context.Context,
	ticketID string,
	db dbExecutor,
) (entities.OpsBooking, error) {
	var payload []byte

	err := db.QueryRowContext(
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

type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
