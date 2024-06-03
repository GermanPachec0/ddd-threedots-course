package outbox

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
)

func NewPublisherForDb(ctx context.Context, db *sqlx.Tx) (message.Publisher, error) {
	var publisher message.Publisher

	logger := log.NewWatermill(log.FromContext(ctx))

	publisher, err := watermillSQL.NewPublisher(
		db,
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return nil, err
	}
	publisher = log.CorrelationPublisherDecorator{publisher}

	publisher = forwarder.NewPublisher(publisher, forwarder.PublisherConfig{
		ForwarderTopic: topic,
	})
	return publisher, nil
}
