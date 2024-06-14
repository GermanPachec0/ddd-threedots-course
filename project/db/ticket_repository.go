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
	Update(ctx context.Context, ticket entities.Ticket) error
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

func (tr TicketRepository) Update(ctx context.Context, ticket entities.Ticket) error {
	var exists bool
	err := tr.db.Conn.GetContext(ctx, &exists, `
        SELECT EXISTS (SELECT 1 FROM tickets WHERE ticket_id = $1)`, ticket.TicketID)
	if err != nil {
		return fmt.Errorf("could not check if ticket exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("ticket with id %s does not exist", ticket.TicketID)
	}

	_, err = tr.db.Conn.NamedExecContext(ctx, `
        UPDATE tickets SET deleted_at = :deleted_at WHERE ticket_id = :ticket_id`, ticket)
	if err != nil {
		return fmt.Errorf("could not update ticket: %w", err)
	}

	return nil
}
func (tr TicketRepository) Get(ctx context.Context) ([]entities.Ticket, error) {
	var tickets []entities.Ticket
	err := tr.db.Conn.SelectContext(ctx, &tickets, `
    SELECT ticket_id, 
           price_amount AS "price.amount",
           price_currency AS "price.currency", 
           customer_email
    FROM tickets 
    WHERE tickets.deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("could not get all the tickets %w", err)
	}

	return tickets, nil
}
