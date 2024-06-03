// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
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

	err = postgresSub.SubscribeInitialize("ItemAddedToCart")
	require.NoError(t, err)

	messages, err := postgresSub.Subscribe(context.Background(), "ItemAddedToCart")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)

	msgToPublish := watermillMessage.NewMessage(uuid.NewString(), []byte(`{"item_id": "1"}`))
	msgToPublish.Metadata.Set("event_type", "ItemAddedToCart")

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
