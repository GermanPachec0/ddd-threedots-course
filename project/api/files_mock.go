package api

import (
	"context"
	"sync"
)

type FileServiceClientMock struct {
	mock sync.Mutex
}

func (c *FileServiceClientMock) StoreFile(ctx context.Context, ticketFile string, ticketHTML string) error {
	c.mock.Lock()
	defer c.mock.Lock()

	return nil
}
