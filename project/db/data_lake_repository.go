package db

import (
	"context"
	"encoding/json"
	"fmt"
	"tickets/entities"
)

type IEventRepository interface {
	Create(ctx context.Context, event entities.Event) error
}

type EventRepository struct {
	db *DB
}

func NewEventRepository(db *DB) EventRepository {
	if db == nil {
		panic("db is nil")
	}
	return EventRepository{
		db: db,
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
