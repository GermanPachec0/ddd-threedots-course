package db

import (
	"context"
	"tickets/entities"
)

type TicketRepositoryMock struct {
}

func NewTicketRepoMock() TicketRepositoryMock {
	return TicketRepositoryMock{}
}

func (tr TicketRepositoryMock) Create(ctx context.Context, ticket entities.TicketBookingConfirmed) error {
	return nil
}
