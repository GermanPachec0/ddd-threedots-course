package orders

import (
	"os"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

func Initialize(
	e *echo.Echo,
	commandBus *cqrs.CommandBus,
	commandProcessor *cqrs.CommandProcessor,
	eventBus *cqrs.EventBus,
	eventProcessor *cqrs.EventProcessor,
) {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}

	initializeDatabaseSchema(db)

	mountHttpHandlers(e, db)
}
