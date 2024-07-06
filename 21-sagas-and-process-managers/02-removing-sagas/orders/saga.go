package orders

import (
	"context"
	"remove_sagas/common"
)

type CommandBus interface {
	Send(ctx context.Context, command any) error
}

type OrderSaga struct {
	commandBus CommandBus
}

func NewOrderSaga(commandBus CommandBus) *OrderSaga {
	return &OrderSaga{commandBus: commandBus}
}

func (o *OrderSaga) HandleOrderPlaced(ctx context.Context, event *common.OrderPlaced) error {
	return o.commandBus.Send(ctx, common.RemoveProductsFromStock{
		OrderID:  event.OrderID,
		Products: event.Products,
	})
}

func (o *OrderSaga) HandleProductsRemovedFromStock(ctx context.Context, event *common.ProductsRemovedFromStock) error {
	return o.commandBus.Send(ctx, common.ShipOrder{
		OrderID: event.OrderID,
	})
}

func (o *OrderSaga) HandleProductsOutOfStock(ctx context.Context, event *common.ProductsOutOfStock) error {
	return o.commandBus.Send(ctx, common.CancelOrder{
		OrderID: event.OrderID,
	})
}
