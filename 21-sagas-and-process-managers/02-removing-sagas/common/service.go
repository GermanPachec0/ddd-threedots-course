package common

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type AddHandlersFn func(
	echo *echo.Echo,
	commandBus *cqrs.CommandBus,
	commandProcessor *cqrs.CommandProcessor,
	eventBus *cqrs.EventBus,
	eventProcessor *cqrs.EventProcessor,
)

func StartService(ctx context.Context, addMessageHandlers []AddHandlersFn) {
	log.Init(logrus.InfoLevel)

	watermillLogger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	var pub message.Publisher
	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, watermillLogger)
	if err != nil {
		panic(err)
	}

	pub = PublisherDecorator{pub}

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		panic(err)
	}

	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			topic := message.SubscribeTopicFromCtx(msg.Context())
			handler := message.HandlerNameFromCtx(msg.Context())

			msgs, err := h(msg)

			logrus.WithFields(logrus.Fields{
				"message_uuid": msg.UUID,
				"topic":        topic,
				"handler":      handler,
				"err":          err,
			}).Info("Message processed")

			return msgs, err
		}
	})

	eventBus := NewEventBus(pub)

	eventProcessorConfig := NewEventProcessorConfig(rdb, watermillLogger)
	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, eventProcessorConfig)
	if err != nil {
		panic(err)
	}

	commandBus := NewCommandBus(pub, watermillLogger)

	commandProcessor, err := cqrs.NewCommandProcessorWithConfig(router, NewCommandProcessorConfig(rdb, watermillLogger))
	if err != nil {
		panic(err)
	}

	e := commonHTTP.NewEcho()

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	for _, addHandlerFn := range addMessageHandlers {
		addHandlerFn(e, commandBus, commandProcessor, eventBus, eventProcessor)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	errgrp := errgroup.Group{}

	errgrp.Go(func() error {
		return router.Run(ctx)
	})

	errgrp.Go(func() error {
		return e.Start(":8080")
	})

	errgrp.Go(func() error {
		<-ctx.Done()
		return e.Shutdown(ctx)
	})

	if err = errgrp.Wait(); err != nil {
		panic(err)
	}
}

type PublisherDecorator struct {
	pub message.Publisher
}

func (p PublisherDecorator) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		logrus.WithFields(logrus.Fields{
			"message_uuid": msg.UUID,
			"topic":        topic,
			"name":         msg.Metadata.Get("name"),
		}).Info("Publishing message")
	}

	return p.pub.Publish(topic, messages...)
}

func (p PublisherDecorator) Close() error {
	return p.pub.Close()
}

func NewEventBus(pub message.Publisher) *cqrs.EventBus {
	eventBus, err := cqrs.NewEventBusWithConfig(
		pub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return "events." + params.EventName, nil
			},
			Marshaler: cqrs.JSONMarshaler{
				GenerateName: cqrs.StructName,
			},
		},
	)
	if err != nil {
		panic(err)
	}

	return eventBus
}

func NewEventProcessorConfig(redisClient *redis.Client, watermillLogger watermill.LoggerAdapter) cqrs.EventProcessorConfig {
	return cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			return "events." + params.EventName, nil
		},
		SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return redisstream.NewSubscriber(redisstream.SubscriberConfig{
				Client:        redisClient,
				ConsumerGroup: "events." + params.HandlerName,
			}, watermillLogger)
		},
		Marshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
		Logger: watermillLogger,
	}
}

func NewCommandBus(publisher message.Publisher, watermillLogger watermill.LoggerAdapter) *cqrs.CommandBus {
	commandBus, err := cqrs.NewCommandBusWithConfig(publisher, cqrs.CommandBusConfig{
		GeneratePublishTopic: func(params cqrs.CommandBusGeneratePublishTopicParams) (string, error) {
			return "commands." + params.CommandName, nil
		},
		Marshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
		Logger: watermillLogger,
	})
	if err != nil {
		panic(err)
	}

	return commandBus
}

func NewCommandProcessorConfig(
	redisClient *redis.Client,
	watermillLogger watermill.LoggerAdapter,
) cqrs.CommandProcessorConfig {
	return cqrs.CommandProcessorConfig{
		SubscriberConstructor: func(params cqrs.CommandProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return redisstream.NewSubscriber(
				redisstream.SubscriberConfig{
					Client:        redisClient,
					ConsumerGroup: "commands." + params.HandlerName,
				},
				watermillLogger,
			)
		},
		GenerateSubscribeTopic: func(params cqrs.CommandProcessorGenerateSubscribeTopicParams) (string, error) {
			return "commands." + params.CommandName, nil
		},
		Marshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
		Logger: watermillLogger,
	}
}
