package events

import (
	"context"
	"log/slog"
	"net/http"
	"tickets/adapters/echo_server"
	"tickets/adapters/rds"
	"tickets/pkg/receipts"
	"tickets/pkg/sheets"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/lithammer/shortuuid/v3"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Router struct {
	router      *message.Router
	server      *echo_server.Server
	redisClient *redis.Client
	clients     *clients.Clients
}

func NewRouter(redisClient *redis.Client, receiptClient receipts.ReceiptService, spreedShetClient sheets.SpreedSheetService) (*Router, error) {
	router, err := newRouter()
	if err != nil {
		slog.Info("Erorr creating router")
		return nil, err
	}

	handler := NewHandler(spreedShetClient, receiptClient)

	logger := watermill.NewStdLogger(false, false)
	receiptSub, err := rds.NewSubscriber(redisClient, "append-to-tracker", logger)

	router.AddNoPublisherHandler(
		"TicketBookingConfirmed",
		"TicketBookingConfirmed",
		receiptSub,
		handler.HandleTicketConfirmed,
	)

	router.AddNoPublisherHandler(
		"TicketBookingCanceled",
		"TicketBookingCanceled",
		receiptSub,
		handler.HandleTicketCancel,
	)

	issueReceiptSub, err := rds.NewSubscriber(redisClient, "issue-receipt", logger)
	if err != nil {
		slog.Error("error creating issue receipt subscriber in redis")
		return nil, err
	}

	router.AddNoPublisherHandler(
		"issue-queue",
		"TicketBookingConfirmed",
		issueReceiptSub,
		handler.HandleTicketIssued,
	)

	ticketPublisher, err := rds.NewRedisPublisher(redisClient, logger)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	slog := slog.Default()

	server := echo_server.NewServer("8080", ticketPublisher, slog)

	return &Router{
		router: router,
		server: server,
	}, nil
}

func (r *Router) Run(ctx context.Context) error {
	slog.Info("Starting router")
	rg, ctx := errgroup.WithContext(ctx)
	rg.Go(func() error {
		err := r.router.Run(ctx)
		return err
	})
	rg.Go(func() error {
		err := r.server.Start()
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
	rg.Go(func() error {
		<-r.router.Running()
		slog.Info("Starting the webhook")

		return nil
	})
	rg.Go(func() error {
		<-ctx.Done()
		return r.server.Stop()
	})

	return rg.Wait()
}

func newRouter() (*message.Router, error) {
	watermillLogger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	m := middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          watermillLogger,
	}
	router.AddMiddleware(PropagateCorrelationID)
	router.AddMiddleware(LogMessage)
	router.AddMiddleware(m.Middleware)

	return router, nil

}

func PropagateCorrelationID(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		correlationID := msg.Metadata.Get("correlation_id")
		if correlationID == "" {
			correlationID = shortuuid.New()
		}

		ctx := log.ToContext(msg.Context(), logrus.WithFields(logrus.Fields{"correlation_id": correlationID}))
		ctx = log.ContextWithCorrelationID(ctx, correlationID)

		msg.SetContext(ctx)
		return next(msg)
	}
}

func LogMessage(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		logger := log.FromContext(msg.Context())
		correlationID := log.CorrelationIDFromContext(msg.Context())
		logger.Info("Handling a message")
		logger = logger.WithField("message_uuid", correlationID)
		msgs, err := next(msg)
		if err != nil {
			logger.WithField("error", err.Error()).Error("Message handling error")
		}

		return msgs, nil
	}
}
