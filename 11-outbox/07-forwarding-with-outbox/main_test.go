// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var outboxTopic = "events_to_forward"

func Test(t *testing.T) {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	require.NoError(t, err)

	watermillLogger := watermill.NewStdLogger(true, true)

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	require.NoError(t, err)

	err = RunForwarder(db, rdb, outboxTopic, watermillLogger)
	require.NoError(t, err)

	redisSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client: rdb,
	}, watermillLogger)

	messages, err := redisSub.Subscribe(context.Background(), "ItemAddedToCart")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)

	msgToPublish := watermillMessage.NewMessage(uuid.NewString(), []byte("1234"))

	err = PublishInTx(msgToPublish, tx, watermillLogger)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	select {
	case <-time.After(time.Second * 5):
		t.Fatal("timeout waiting for message")
	case msg := <-messages:
		assert.Equal(t, msgToPublish.UUID, msg.UUID)
		assert.Equal(t, msgToPublish.Payload, msg.Payload)
	}
}

func PublishInTx(
	msg *message.Message,
	tx *sql.Tx,
	logger watermill.LoggerAdapter,
) error {
	var publisher message.Publisher
	var err error

	publisher, err = watermillSQL.NewPublisher(
		tx,
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox publisher: %w", err)
	}

	publisher = forwarder.NewPublisher(publisher, forwarder.PublisherConfig{
		ForwarderTopic: outboxTopic,
	})

	return publisher.Publish("ItemAddedToCart", msg)
}
