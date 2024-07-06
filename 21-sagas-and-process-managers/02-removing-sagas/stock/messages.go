package stock

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"remove_sagas/common"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func mountMessageHandlers(db *sqlx.DB, eventBus *cqrs.EventBus, commandProcessor *cqrs.CommandProcessor) {
	h := Handler{db, eventBus}

	err := commandProcessor.AddHandlers(
		cqrs.NewCommandHandler(
			"RemoveProductsFromStock",
			h.RemoveProductsFromStock,
		),
	)
	if err != nil {
		panic(err)
	}
}

type Handler struct {
	db       *sqlx.DB
	eventBus *cqrs.EventBus
}

var ProductsOutOfStockError = fmt.Errorf("products out of stock")

func (h Handler) RemoveProductsFromStock(ctx context.Context, cmd *common.RemoveProductsFromStock) error {
	missingProducts := make(map[uuid.UUID]int)

	err := common.UpdateInTx(
		ctx,
		h.db,
		sql.LevelSerializable,
		func(ctx context.Context, tx *sqlx.Tx) error {
			for productID, quantity := range cmd.Products {
				quantityInStock := 0

				err := tx.Get(
					&quantityInStock,
					"SELECT quantity FROM stock WHERE product_id = $1",
					productID,
				)
				if err != nil {
					return err
				}

				if quantityInStock < quantity {
					missingProducts[productID] = quantity - quantityInStock
				}

				if len(missingProducts) > 0 {
					continue
				}

				_, err = tx.Exec(
					"UPDATE stock SET quantity = quantity - $1 WHERE product_id = $2",
					quantity,
					productID,
				)
				if err != nil {
					return err
				}
			}

			if len(missingProducts) > 0 {
				return ProductsOutOfStockError
			}

			return nil
		},
	)

	if errors.Is(err, ProductsOutOfStockError) {
		// we should use outbox here, but I don't want to add you more stuff to remove
		return h.eventBus.Publish(ctx, &common.ProductsOutOfStock{
			OrderID:         cmd.OrderID,
			MissingProducts: missingProducts,
		})
	}
	if err != nil {
		return err
	}

	// we should use outbox here, but I don't want to add you more stuff to remove
	return h.eventBus.Publish(ctx, &common.ProductsRemovedFromStock{
		OrderID:  cmd.OrderID,
		Products: cmd.Products,
	})
}
