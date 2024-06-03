package db

import (
	"context"
	"fmt"
	"tickets/entities"

	"github.com/google/uuid"
)

type IShowRepository interface {
	Create(ctx context.Context, show entities.Show) (entities.ShowCreateResponse, error)
}

type ShowRepository struct {
	db *DB
}

func NewShowRepository(db *DB) ShowRepository {
	if db == nil {
		panic("db is nil")
	}
	return ShowRepository{
		db: db,
	}
}

func (tr ShowRepository) Create(ctx context.Context, show entities.Show) (entities.ShowCreateResponse, error) {
	var showID uuid.UUID

	err := tr.db.Conn.QueryRowContext(
		ctx,
		`
		INSERT INTO shows (dead_nation_id, number_of_tickets, start_time, title, venue) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING show_id`,
		show.DeadNationID, show.NumberOfTickets, show.StartTime, show.Title, show.Venue,
	).Scan(&showID)

	if err != nil {
		return entities.ShowCreateResponse{}, fmt.Errorf("could not save show: %w", err)
	}

	return entities.ShowCreateResponse{ShowID: showID}, nil

}

func (tr ShowRepository) ShowByID(ctx context.Context, showID uuid.UUID) (entities.Show, error) {
	var show entities.Show
	err := tr.db.Conn.GetContext(ctx, &show, `
		SELECT 
		    * 
		FROM 
		    shows
		WHERE
		    show_id = $1
	`, showID)
	if err != nil {
		return entities.Show{}, fmt.Errorf("could not get show: %w", err)
	}

	return show, nil
}
