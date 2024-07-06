package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"tickets/entities"
	"tickets/message/event"
	"tickets/message/outbox"
	"tickets/message/sagas"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type VipBundleRepository struct {
	db *sqlx.DB
}

func NewVipBundleRepository(db *sqlx.DB) *VipBundleRepository {
	if db == nil {
		panic("db must be set")
	}

	return &VipBundleRepository{db: db}
}

type Executor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func (v VipBundleRepository) Add(ctx context.Context, vipBundle sagas.VipBundle) error {
	payload, err := json.Marshal(vipBundle)
	if err != nil {
		return fmt.Errorf("could not marshal vip bundle: %w", err)
	}

	return updateInTx(
		ctx,
		v.db,
		sql.LevelRepeatableRead,
		func(ctx context.Context, tx *sqlx.Tx) error {
			_, err = v.db.ExecContext(ctx, `
				INSERT INTO vip_bundles (vip_bundle_id, booking_id, payload)
				VALUES ($1, $2, $3)
			`, vipBundle.VipBundleID, vipBundle.BookingID, payload)

			if err != nil {
				return fmt.Errorf("could not insert vip bundle: %w", err)
			}

			outboxPublisher, err := outbox.NewPublisherForDb(ctx, tx)
			if err != nil {
				return fmt.Errorf("could not create event bus: %w", err)
			}

			err = event.NewBus(outboxPublisher).Publish(ctx, entities.VipBundleInitialized_v1{
				Header:      entities.NewEventHeader(),
				VipBundleID: vipBundle.VipBundleID,
			})
			if err != nil {
				return fmt.Errorf("could not publish event: %w", err)
			}

			return nil
		},
	)
}

func (v VipBundleRepository) Get(ctx context.Context, vipBundleID uuid.UUID) (sagas.VipBundle, error) {
	return v.vipBundleByID(ctx, vipBundleID, v.db)
}

func (v VipBundleRepository) vipBundleByID(ctx context.Context, vipBundleID uuid.UUID, db Executor) (sagas.VipBundle, error) {
	var payload []byte
	err := v.db.QueryRowContext(ctx, `
		SELECT payload FROM vip_bundles WHERE vip_bundle_id = $1
	`, vipBundleID).Scan(&payload)

	if err != nil {
		return sagas.VipBundle{}, fmt.Errorf("could not get vip bundle: %w", err)
	}

	var vipBundle sagas.VipBundle
	err = json.Unmarshal(payload, &vipBundle)
	if err != nil {
		return sagas.VipBundle{}, fmt.Errorf("could not unmarshal vip bundle: %w", err)
	}

	return vipBundle, nil
}

func (v VipBundleRepository) GetByBookingID(ctx context.Context, bookingID uuid.UUID) (sagas.VipBundle, error) {
	return v.getByBookingID(ctx, bookingID, v.db)
}

func (v VipBundleRepository) getByBookingID(ctx context.Context, bookingID uuid.UUID, db Executor) (sagas.VipBundle, error) {
	var payload []byte
	err := db.QueryRowContext(ctx, `
		SELECT payload FROM vip_bundles WHERE booking_id = $1
	`, bookingID).Scan(&payload)

	if err != nil {
		return sagas.VipBundle{}, fmt.Errorf("could not get vip bundle: %w", err)
	}

	var vipBundle sagas.VipBundle
	err = json.Unmarshal(payload, &vipBundle)
	if err != nil {
		return sagas.VipBundle{}, fmt.Errorf("could not unmarshal vip bundle: %w", err)
	}

	return vipBundle, nil
}

func (v VipBundleRepository) UpdateByID(ctx context.Context, bookingID uuid.UUID, updateFn func(vipBundle sagas.VipBundle) (sagas.VipBundle, error)) (sagas.VipBundle, error) {
	var vb sagas.VipBundle

	err := updateInTx(ctx, v.db, sql.LevelSerializable, func(ctx context.Context, tx *sqlx.Tx) error {
		var err error
		vb, err = v.vipBundleByID(ctx, bookingID, tx)
		if err != nil {
			return err
		}

		vb, err = updateFn(vb)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(vb)
		if err != nil {
			return fmt.Errorf("could not marshal vip bundle: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE vip_bundles SET payload = $1 WHERE vip_bundle_id = $2
		`, payload, vb.VipBundleID)

		if err != nil {
			return fmt.Errorf("could not update vip bundle: %w", err)
		}

		return nil
	})
	if err != nil {
		return sagas.VipBundle{}, fmt.Errorf("could not update vip bundle: %w", err)
	}

	return vb, nil
}

func (v VipBundleRepository) UpdateByBookingID(ctx context.Context, bookingID uuid.UUID, updateFn func(vipBundle sagas.VipBundle) (sagas.VipBundle, error)) (sagas.VipBundle, error) {
	var vb sagas.VipBundle

	err := updateInTx(ctx, v.db, sql.LevelSerializable, func(ctx context.Context, tx *sqlx.Tx) error {
		var err error
		vb, err = v.getByBookingID(ctx, bookingID, tx)
		if err != nil {
			return err
		}

		vb, err = updateFn(vb)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(vb)
		if err != nil {
			return fmt.Errorf("could not marshal vip bundle: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE vip_bundles SET payload = $1 WHERE booking_id = $2
		`, payload, vb.BookingID)

		if err != nil {
			return fmt.Errorf("could not update vip bundle: %w", err)
		}

		return nil
	})
	if err != nil {
		return sagas.VipBundle{}, fmt.Errorf("could not update vip bundle: %w", err)
	}

	return vb, nil
}
