package common

import "github.com/google/uuid"

type OrderPlaced struct {
	OrderID uuid.UUID `json:"order_id"`

	Products map[uuid.UUID]int `json:"products"`
}

type ProductsRemovedFromStock struct {
	OrderID  uuid.UUID         `json:"order_id"`
	Products map[uuid.UUID]int `json:"products"`
}

type ProductsOutOfStock struct {
	OrderID         uuid.UUID         `json:"order_id"`
	MissingProducts map[uuid.UUID]int `json:"missing_products"`
}
