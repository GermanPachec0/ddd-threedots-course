package db

import (
	"database/sql"

	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type TracingPublisherDecorator struct {
	message.Publisher
}

func (s TracingPublisherDecorator) Publish(topic string, messages ...*message.Message) error {
	for i, _ := range messages {
		otel.GetTextMapPropagator().Inject(messages[i].Context(), propagation.MapCarrier(messages[i].Metadata))
	}

	return s.Publisher.Publish(topic, messages...)
}

func PublishInTx(
	message *message.Message,
	tx *sql.Tx,
	logger watermill.LoggerAdapter,
) error {
	publisher, err := watermillSQL.NewPublisher(
		tx,
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return err
	}

	pubTraceDecorator := TracingPublisherDecorator{Publisher: publisher}

	return pubTraceDecorator.Publish("BookingMade", message)
}
