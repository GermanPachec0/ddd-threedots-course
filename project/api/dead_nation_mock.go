package api

import (
	"context"
	"sync"
	"tickets/entities"
)

type DeadNationMock struct {
	mock sync.Mutex
}

func (c *DeadNationMock) CreateBooking(ctx context.Context, booking entities.DeadNationBookingRequest) error {
	c.mock.Lock()
	defer c.mock.Lock()

	return nil
}
