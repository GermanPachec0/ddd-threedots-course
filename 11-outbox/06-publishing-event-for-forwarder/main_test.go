// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
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

func Test(t *testing.T) {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	require.NoError(t, err)

	watermillLogger := watermill.NewStdLogger(true, true)

	postgresSub, err := watermillSQL.NewSubscriber(
		db,
		watermillSQL.SubscriberConfig{
			SchemaAdapter:  watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter: watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
		},
		watermillLogger,
	)
	require.NoError(t, err)

	err = postgresSub.SubscribeInitialize(outboxTopic)
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	require.NoError(t, err)

	redisPub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, watermillLogger)
	require.NoError(t, err)

	redisSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client: rdb,
	}, watermillLogger)

	fwd, err := forwarder.NewForwarder(
		postgresSub,
		redisPub,
		watermillLogger,
		forwarder.Config{
			ForwarderTopic: outboxTopic,
			Middlewares: []message.HandlerMiddleware{
				func(h message.HandlerFunc) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						fmt.Println("Forwarding message", msg.UUID, string(msg.Payload), msg.Metadata)

						return h(msg)
					}
				},
			},
		},
	)
	require.NoError(t, err)

	go func() {
		err := fwd.Run(context.Background())
		require.NoError(t, err)
	}()

	<-fwd.Running()

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
