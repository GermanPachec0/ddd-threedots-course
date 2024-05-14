// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, watermill.NewStdLogger(false, false))
	require.NoError(t, err)

	notificationsClient := &notificationsClient{}
	spreadsheetsClient := &spreadsheetsClient{}

	err = Subscribe(notificationsClient, spreadsheetsClient)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	orderID1 := uuid.NewString()
	orderID2 := uuid.NewString()

	err = pub.Publish("orders-placed", message.NewMessage(watermill.NewUUID(), []byte(orderID1)))
	require.NoError(t, err)

	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		require.Len(t, notificationsClient.orders, 1)
		require.Len(t, spreadsheetsClient.orders, 1)

		assert.Equal(t, orderID1, notificationsClient.orders[0])
		assert.Equal(t, orderID1, spreadsheetsClient.orders[0])
	}, time.Millisecond*500, time.Millisecond*50)

	err = pub.Publish("orders-placed", message.NewMessage(watermill.NewUUID(), []byte(orderID2)))
	require.NoError(t, err)

	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		require.Len(t, notificationsClient.orders, 2)
		require.Len(t, spreadsheetsClient.orders, 2)

		assert.Equal(t, orderID2, notificationsClient.orders[1])
		assert.Equal(t, orderID2, spreadsheetsClient.orders[1])
	}, time.Millisecond*500, time.Millisecond*50)

	groups, err := rdb.XInfoGroups(context.Background(), "orders-placed").Result()
	require.NoError(t, err)

	foundGroups := []string{}
	for _, group := range groups {
		foundGroups = append(foundGroups, group.Name)
	}

	assert.Contains(t, foundGroups, "notifications", "Expected to find notifications consumer group")
	assert.Contains(t, foundGroups, "spreadsheets", "Expected to find spreadsheets consumer group")
}

type notificationsClient struct {
	orders []string
}

func (n *notificationsClient) SendOrderConfirmation(orderID string) error {
	n.orders = append(n.orders, orderID)
	return nil
}

type spreadsheetsClient struct {
	orders []string
}

func (s *spreadsheetsClient) AppendOrderRow(orderID string) error {
	s.orders = append(s.orders, orderID)

	return nil
}
