package stock

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

func mountHttpHandlers(e *echo.Echo, db *sqlx.DB) *echo.Route {
	return e.POST("/products-stock", func(c echo.Context) error {
		productStock := ProductStock{}
		if err := c.Bind(&productStock); err != nil {
			return err
		}
		if productStock.Quantity <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "quantity must be greater than 0")
		}
		if productStock.ProductID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "product_id must be provided")
		}

		_, err := db.Exec(`
			INSERT INTO stock (product_id, quantity)
			VALUES ($1, $2)
			ON CONFLICT (product_id) DO UPDATE SET quantity = stock.quantity + $2
		`, productStock.ProductID, productStock.Quantity)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	})
}
