package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type FileServiceClient struct {
	clients *clients.Clients
}

func NewFileServiceClient(clients *clients.Clients) FileServiceClient {
	return FileServiceClient{
		clients: clients,
	}
}

func (fs FileServiceClient) StoreFile(ctx context.Context, ticketFile string, ticketHTML string) error {

	resp, err := fs.clients.Files.PutFilesFileIdContentWithTextBodyWithResponse(ctx, ticketFile, ticketHTML)
	if err != nil {
		return fmt.Errorf("error saving file %w", err)
	}

	if resp.StatusCode() == http.StatusConflict {
		log.FromContext(ctx).Infof("file %s already exists", ticketFile)
		return nil
	}

	return nil
}
