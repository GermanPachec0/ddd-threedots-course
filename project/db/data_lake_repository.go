package db

import (
	"context"
	"encoding/json"
	"fmt"
	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
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

func (e EventRepository) Create(ctx context.Context, event entities.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling event %w", err)
	}
	_, err = e.db.Conn.ExecContext(ctx, `
		INSERT INTO 
		    events (event_id, published_at, event_name, event_payload)
		VALUES
			 ($1, $2, $3, $4)
		ON CONFLICT (event_id) DO NOTHING; 
`, event.EventID, event.Header.PublishedAt, event.EventName, payload)

	if err != nil {
		return fmt.Errorf("could not create event into db: %w", err)
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
