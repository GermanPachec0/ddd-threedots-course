package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/spreadsheets"
)

type SpreadsheetsAPIClient struct {
	clients *clients.Clients
}

func NewSpreadsheetsAPIClient(clients *clients.Clients) *SpreadsheetsAPIClient {
	if clients == nil {
		panic("New Spread sheets clients is nil")
	}

	return &SpreadsheetsAPIClient{clients: clients}
}

func (c SpreadsheetsAPIClient) AppendRow(ctx context.Context, spreedsheetName string, row []string) error {
	resp, err := c.clients.Spreadsheets.PostSheetsSheetRowsWithResponse(ctx, spreedsheetName, spreadsheets.PostSheetsSheetRowsJSONRequestBody{
		Columns: row,
	})

	if err != nil {
		return fmt.Errorf("Failed to post row %w:", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to post row: unexpected status code %d", resp.StatusCode())
	}
	return nil
}
