package service

import (
	"context"
	"tickets/db"
	ticketsHttp "tickets/http"
	"tickets/message"
	"tickets/message/command"
	"tickets/message/event"
	"tickets/message/outbox"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func init() {
	log.Init(logrus.InfoLevel)
}

type Service struct {
	watermillRouter *watermillMessage.Router
	echoRouter      *echo.Echo
}

func New(
	redisClient *redis.Client,
	spreadsheetsService event.SpreadsheetsAPI,
	receiptsService event.ReceiptsService,
	fileService event.FileService,
	conn db.DB,
	deadNotionService event.DeadNationService,
) Service {
	watermillLogger := log.NewWatermill(log.FromContext(context.Background()))

	var redisPublisher watermillMessage.Publisher
	redisPublisher = message.NewRedisPublisher(redisClient, watermillLogger)
	redisPublisher = log.CorrelationPublisherDecorator{Publisher: redisPublisher}

	eventBus := event.NewBus(redisPublisher)
	commandBus := command.NewCommandBus(redisPublisher)

	ticketRepo := db.NewTicketRepo(&conn)
	showRepo := db.NewShowRepository(&conn)
	bookingRepo := db.NewBookingRespository(&conn)
	showRepository := db.NewShowRepository(&conn)

	eventsHandler := event.NewHandler(
		spreadsheetsService,
		receiptsService,
		ticketRepo,
		fileService,
		eventBus,
		deadNotionService,
		showRepository,
	)
	commandsHandler := command.NewHandler(eventBus, receiptsService)

	redisSubscriber := message.NewRedisSubscriber(redisClient, watermillLogger)
	eventProcessorConfig := event.NewProcessorConfig(redisClient, watermillLogger)
	commandProccessorConfig := command.NewCommandProcessorConfig(redisClient, watermillLogger)
	opsReadModel := db.NewOpsBookingReadModel(&conn)
	dataLakeRepo := db.NewEventRepository(&conn)

	pgSubscriber := outbox.SubscribeForPGMessages(conn.Conn, watermillLogger)
	watermillRouter := message.NewWatermillRouter(
		pgSubscriber,
		redisSubscriber,
		commandProccessorConfig,
		redisPublisher,
		eventProcessorConfig,
		commandsHandler,
		eventsHandler,
		opsReadModel,
		dataLakeRepo,
		watermillLogger,
	)

	echoRouter := ticketsHttp.NewHttpRouter(
		eventBus,
		commandBus,
		spreadsheetsService,
		ticketRepo,
		showRepo,
		bookingRepo,
		opsReadModel,
	)

	return Service{
		watermillRouter,
		echoRouter,
	}
}

func (s Service) Run(
	ctx context.Context,
) error {
	errgrp, ctx := errgroup.WithContext(ctx)

	errgrp.Go(func() error {
		return s.watermillRouter.Run(ctx)
	})

	errgrp.Go(func() error {
		// we don't want to start HTTP server before Watermill router (so service won't be healthy before it's ready)
		<-s.watermillRouter.Running()

		err := s.echoRouter.Start(":8080")

		if err != nil {
			return err
		}

		return nil
	})

	errgrp.Go(func() error {
		<-ctx.Done()
		return s.echoRouter.Shutdown(context.Background())
	})

	return errgrp.Wait()
}
