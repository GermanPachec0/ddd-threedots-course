package entities

import "github.com/google/uuid"

type InternalOpsReadModelUpdated struct {
	Header EventHeader `json:"header"`

	BookingID uuid.UUID `json:"booking_id"`
}

func (i InternalOpsReadModelUpdated) IsInternal() bool {
	return true
}
