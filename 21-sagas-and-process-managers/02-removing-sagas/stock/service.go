package stock

import (
	"os"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

func Initialize(
	e *echo.Echo,
	_ *cqrs.CommandBus,
	commandProcessor *cqrs.CommandProcessor,
	eventBus *cqrs.EventBus,
	_ *cqrs.EventProcessor,
) {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}

	initializeDatabaseSchema(db)

	mountHttpHandlers(e, db)
	mountMessageHandlers(db, eventBus, commandProcessor)
}
