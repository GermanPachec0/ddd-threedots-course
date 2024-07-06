package db

import (
	"context"
	"errors"
	"fmt"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/lib/pq"
)

type IEventRepository interface {
	Create(ctx context.Context, event entities.Event) error
	GetAll(ctx context.Context) ([]entities.Event, error)
}

type EventRepository struct {
	db       *DB
	eventBus *cqrs.EventBus
}

func NewEventRepository(db *DB, eventBus *cqrs.EventBus) EventRepository {
	if db == nil {
		panic("db is nil")
	}
	return EventRepository{
		db:       db,
		eventBus: eventBus,
	}
}

func (s EventRepository) Create(
	ctx context.Context,
	dataLakeEvent entities.Event,
) error {
	_, err := s.db.Conn.NamedExecContext(
		ctx,
		`
			INSERT INTO 
			    events (event_id, published_at, event_name, event_payload) 
			VALUES 
			    (:event_id, :published_at, :event_name, :event_payload)`,
		dataLakeEvent,
	)
	var postgresError *pq.Error
	if errors.As(err, &postgresError) && postgresError.Code.Name() == "unique_violation" {
		// handling re-delivery
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not store %s event in data lake: %w", dataLakeEvent.EventID, err)
	}

	return nil
}

func (e EventRepository) GetAll(ctx context.Context) ([]entities.Event, error) {
	var events []entities.Event
	err := e.db.Conn.SelectContext(ctx, &events, "SELECT * FROM events ORDER BY published_at ASC")
	if err != nil {
		return nil, fmt.Errorf("error getting all events %w", err)
	}
	return events, nil
}
