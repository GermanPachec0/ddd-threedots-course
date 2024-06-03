// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	require.NoError(t, err)

	watermillLogger := watermill.NewStdLogger(true, true)

	payload, err := json.Marshal(map[string]any{
		"item_id": rand.Intn(100),
	})
	require.NoError(t, err)

	msgToPublish := watermillMessage.NewMessage(uuid.NewString(), payload)

	msgs, err := SubscribeForMessages(db, "ItemAddedToCart", watermillLogger)
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)

	err = PublishInTx(msgToPublish, tx, watermillLogger)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	select {
	case <-time.After(time.Second * 5):
		t.Fatal("timeout waiting for message")
	case msg := <-msgs:
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

	return publisher.Publish("ItemAddedToCart", msg)
}
