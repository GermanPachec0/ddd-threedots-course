package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"tickets/adapters/rds"
	"tickets/events"
	"tickets/pkg/receipts"
	"tickets/pkg/sheets"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/sirupsen/logrus"
)

func main() {
	log.Init(logrus.InfoLevel)

	rdb := rds.NewRedisClient(os.Getenv("REDIS_ADDR"))
	clients, err := clients.NewClients(os.Getenv("GATEWAY_ADDR"), func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
		return nil
	})
	if err != nil {
		panic(err)
	}

	spreedSheetSVC := sheets.NewSpreadsheetsClient(clients)
	receiptSVC := receipts.NewReceiptsClient(clients)

	router, err := events.NewRouter(rdb, receiptSVC, spreedSheetSVC)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)

	defer cancel()

	err = router.Run(ctx)
	if err != nil {
		panic(err)
	}
}
