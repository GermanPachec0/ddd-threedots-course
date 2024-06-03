package db

import (
	"context"
	"fmt"
	"tickets/entities"
)

type ITicketRepository interface {
	Create(ctx context.Context, ticket entities.Ticket) error
	Delete(ctx context.Context, ticket entities.Ticket) error
	Get(ctx context.Context) ([]entities.Ticket, error)
}

type TicketRepository struct {
	db *DB
}

func NewTicketRepo(db *DB) TicketRepository {
	if db == nil {
		panic("db is nil")
	}
	return TicketRepository{
		db: db,
	}
}

func (tr TicketRepository) Create(ctx context.Context, ticket entities.Ticket) error {
	_, err := tr.db.Conn.NamedExecContext(
		ctx,
		`
		INSERT INTO 
    		tickets (ticket_id, price_amount, price_currency, customer_email) 
		VALUES 
		    (:ticket_id, :price.amount, :price.currency, :customer_email) ON CONFLICT DO NOTHING`,
		ticket,
	)
	if err != nil {
		return fmt.Errorf("could not save ticket: %w", err)
	}
	return err
}

func (tr TicketRepository) Delete(ctx context.Context, ticket entities.Ticket) error {
	_, err := tr.db.Conn.NamedExecContext(ctx,
		`DELETE FROM tickets WHERE ticket_id = :ticket_id`,
		ticket)
	if err != nil {
		return fmt.Errorf("could not delete canceled tickets %w", err)
	}
	return nil
}

func (tr TicketRepository) Get(ctx context.Context) ([]entities.Ticket, error) {
	var tickets []entities.Ticket
	err := tr.db.Conn.SelectContext(ctx, &tickets, `
	SELECT ticket_id, price_amount as "price.amount",
	price_currency as "price.currency", customer_email
	from tickets`)
	if err != nil {
		return nil, fmt.Errorf("could not get all the tickets %w", err)
	}

	return tickets, nil
}
