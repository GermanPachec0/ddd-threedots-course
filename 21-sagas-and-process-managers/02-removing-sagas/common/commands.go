package common

import "github.com/google/uuid"

type CancelOrder struct {
	OrderID uuid.UUID `json:"order_id"`
}

type RemoveProductsFromStock struct {
	OrderID  uuid.UUID         `json:"order_id"`
	Products map[uuid.UUID]int `json:"products"`
}

type ShipOrder struct {
	OrderID uuid.UUID `json:"order_id"`
}
