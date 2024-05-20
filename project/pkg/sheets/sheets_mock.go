package sheets

import (
	"context"
	"sync"
)

type SpreedSheetServiceMock struct {
	mock sync.Mutex
	Rows map[string][][]string
}

func (ms *SpreedSheetServiceMock) AppendRow(ctx context.Context, sheetName string, row []string) error {
	ms.mock.Lock()
	defer ms.mock.Unlock()
	if ms.Rows == nil {
		ms.Rows = make(map[string][][]string)
	}

	ms.Rows[sheetName] = append(ms.Rows[sheetName], row)
	return nil
}
