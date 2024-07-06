package common

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"
)

type UUIDs []uuid.UUID

func (u *UUIDs) Scan(src any) error {
	stringArray := pq.StringArray{}

	if err := stringArray.Scan(src); err != nil {
		return err
	}

	uuids := make([]uuid.UUID, len(stringArray))
	for _, s := range stringArray {
		uuid, err := uuid.Parse(s)
		if err != nil {
			return fmt.Errorf("failed to parse uuid %s: %w", s, err)
		}
		uuids = append(uuids, uuid)
	}

	*u = uuids
	return nil
}

func (u UUIDs) Value() (driver.Value, error) {
	strUUIDs := lo.Map(u, func(uuid uuid.UUID, _ int) string {
		return uuid.String()
	})

	return pq.StringArray(strUUIDs).Value()
}
