package outbox

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
)

func SubscribeForPGMessages(db *sqlx.DB, logger watermill.LoggerAdapter) message.Subscriber {
	subConfig := sql.SubscriberConfig{
		SchemaAdapter:  sql.DefaultPostgreSQLSchema{},
		OffsetsAdapter: sql.DefaultPostgreSQLOffsetsAdapter{},
	}

	sub, err := sql.NewSubscriber(db, subConfig, logger)
	if err != nil {
		panic(err)
	}
	err = sub.SubscribeInitialize(topic)
	if err != nil {
		panic(err)
	}

	return sub
}
