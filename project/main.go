package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"tickets/api"
	"tickets/db"
	"tickets/message"
	"tickets/service"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	database, err := db.NewDBConn(os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}
	database.MigrateSchema()
	defer database.Close()

	apiClients, err := clients.NewClients(
		os.Getenv("GATEWAY_ADDR"),
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
			return nil
		},
	)
	if err != nil {
		panic(err)
	}

	redisClient := message.NewRedisClient(os.Getenv("REDIS_ADDR"))
	defer redisClient.Close()

	spreadsheetsService := api.NewSpreadsheetsAPIClient(apiClients)
	receiptsService := api.NewReceiptsServiceClient(apiClients)
	fileService := api.NewFileServiceClient(apiClients)
	deadNotionService := api.NewDeadNotionClient(apiClients)

	err = service.New(
		redisClient,
		spreadsheetsService,
		receiptsService,
		fileService,
		database,
		deadNotionService,
	).Run(ctx)
	if err != nil {
		panic(err)
	}
}
