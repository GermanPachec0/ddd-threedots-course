package orders

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"remove_sagas/common"
	"remove_sagas/stock"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type PostOrderRequest struct {
	OrderID   uuid.UUID         `json:"order_id"`
	Products  map[uuid.UUID]int `json:"products"`
	Shipped   bool              `json:"shipped"`
	Cancelled bool              `json:"cancelled"`
}

type GetOrderResponse struct {
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`
	Shipped   bool      `json:"shipped" db:"shipped"`
	Cancelled bool      `json:"cancelled" db:"cancelled"`
}

func mountHttpHandlers(e *echo.Echo, db *sqlx.DB) {
	e.POST("/orders", func(c echo.Context) error {
		order := PostOrderRequest{}
		if err := c.Bind(&order); err != nil {
			return err
		}

		err := common.UpdateInTx(
			c.Request().Context(),
			db,
			sql.LevelSerializable,
			func(ctx context.Context, tx *sqlx.Tx) error {
				for product, quantity := range order.Products {
					var productStock stock.ProductStock
					err := db.Get(
						&productStock,
						"SELECT product_id, quantity FROM stock WHERE product_id = $1",
						product,
					)
					slog.Info("Products: %w", productStock)
					if err != nil {
						slog.Error("error %w", err)
						return err
					}

					if productStock.Quantity < quantity {
						_, err := tx.Exec(
							"INSERT INTO orders (order_id, shipped, cancelled) VALUES ($1, $2, $3)",
							order.OrderID,
							false,
							true,
						)

						if err != nil {
							return err
						}
						return c.JSON(http.StatusCreated, nil)
					}

					_, err = tx.Exec(`
						INSERT INTO stock (product_id, quantity)
						VALUES ($1, $2)
						ON CONFLICT (product_id) DO UPDATE SET quantity = stock.quantity - $2`,
						productStock.ProductID, quantity)
					if err != nil {
						slog.Error("error %w", err)
						return err
					}

					_, err = tx.Exec(
						"INSERT INTO order_products (order_id, product_id, quantity) VALUES ($1, $2, $3)",
						order.OrderID,
						productStock.ProductID,
						quantity,
					)
					if err != nil {
						slog.Error("error %w", err)

						return err
					}
				}
				_, err := tx.Exec(
					"INSERT INTO orders (order_id, shipped, cancelled) VALUES ($1, $2, $3)",
					order.OrderID,
					true,
					false,
				)
				if err != nil {
					slog.Error("error %w", err)

					return err
				}
				return c.NoContent(http.StatusCreated)
			},
		)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	})

	e.GET("/orders/:order_id", func(c echo.Context) error {
		orderID, err := uuid.Parse(c.Param("order_id"))
		if err != nil {
			return err
		}

		order := GetOrderResponse{}

		err = db.Get(
			&order,
			"SELECT order_id, shipped, cancelled FROM orders WHERE order_id = $1",
			orderID,
		)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, PostOrderRequest{
			OrderID:   order.OrderID,
			Shipped:   order.Shipped,
			Cancelled: order.Cancelled,
		})
	})
}
