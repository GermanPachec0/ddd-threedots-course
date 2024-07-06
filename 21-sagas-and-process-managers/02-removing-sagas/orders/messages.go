package orders

import (
	"context"
	"remove_sagas/common"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/jmoiron/sqlx"
)

func mountMessageHandlers(db *sqlx.DB, commandBus *cqrs.CommandBus, eventProcessor *cqrs.EventProcessor, commandProcessor *cqrs.CommandProcessor) {
	orderSaga := NewOrderSaga(commandBus)

	err := eventProcessor.AddHandlers(
		cqrs.NewEventHandler(
			"order_saga.HandleOrderPlaced",
			orderSaga.HandleOrderPlaced,
		),
		cqrs.NewEventHandler(
			"order_saga.HandleProductsRemovedFromStock",
			orderSaga.HandleProductsRemovedFromStock,
		),
		cqrs.NewEventHandler(
			"order_saga.HandleProductsOutOfStock",
			orderSaga.HandleProductsOutOfStock,
		),
	)
	if err != nil {
		panic(err)
	}

	err = commandProcessor.AddHandlers(
		cqrs.NewCommandHandler(
			"ShipOrder",
			func(ctx context.Context, cmd *common.ShipOrder) error {
				_, err := db.Exec("UPDATE orders SET shipped = true WHERE order_id = $1", cmd.OrderID)
				return err
			},
		),
		cqrs.NewCommandHandler(
			"CancelOrder",
			func(ctx context.Context, cmd *common.CancelOrder) error {
				_, err := db.Exec("UPDATE orders SET cancelled = true WHERE order_id = $1", cmd.OrderID)
				return err
			},
		),
	)
	if err != nil {
		panic(err)
	}
}
