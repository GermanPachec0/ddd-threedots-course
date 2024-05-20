package receipts

import (
	"context"
	"sync"
)

type ReceiptsServiceMock struct {
	mock           sync.Mutex
	IssuedReceipts []IssueReceiptRequest
}

func (mr *ReceiptsServiceMock) IssueReceipt(ctx context.Context, ticketRequest IssueReceiptRequest) error {
	mr.mock.Lock()
	mr.IssuedReceipts = append(mr.IssuedReceipts, ticketRequest)
	defer mr.mock.Unlock()
	return nil
}
