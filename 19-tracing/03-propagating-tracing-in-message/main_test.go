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
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func Test(t *testing.T) {
	exp := &tracetest.InMemoryExporter{}

	initTracing(exp)

	logger := watermill.NewStdLogger(true, true)

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: uuid.NewString(),
	}, logger)

	roomBookingConfirmedCh, err := sub.Subscribe(context.Background(), "RoomBookingConfirmed")
	require.NoError(t, err)

	paymentReceivedCh, err := sub.Subscribe(context.Background(), "PaymentReceived")
	require.NoError(t, err)

	router, eventBus := NewRouter(rdb, logger)

	event := PaymentReceived{
		ID:            uuid.NewString(),
		RoomBookingID: uuid.NewString(),
	}

	go func() {
		err := router.Run(context.Background())
		assert.NoError(t, err)
	}()

	<-router.Running()

	ctx, span := otel.Tracer("").Start(context.Background(), "Publish")
	err = eventBus.Publish(ctx, event)
	span.End()

	if err != nil {
		t.Log("Publish error:", err)
	}

	{
		var paymentReceivedChMsg *message.Message

		select {
		case paymentReceivedChMsg = <-paymentReceivedCh:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for PaymentReceived event")
		}

		traceCtx := propagation.TraceContext{}.Extract(
			context.Background(),
			propagation.MapCarrier(paymentReceivedChMsg.Metadata),
		)

		sc := trace.SpanContextFromContext(traceCtx)

		require.True(
			t,
			sc.IsValid(),
			"PaymentReceived has no trace information (should be stored in 'traceparent' metadata): %v. "+
				"Did you decorated publisher?",
			paymentReceivedChMsg.Metadata,
		)

		t.Log("PaymentReceived trace ID:", sc.TraceID().String(), "Metadata:", fmt.Sprintf("%v", paymentReceivedChMsg.Metadata))

		assert.Equal(
			t,
			span.SpanContext().TraceID().String(),
			sc.TraceID().String(),
			"PaymentReceived should maintain trace ID after publishing",
		)
	}

	{
		var roomBookingConfirmedChMsg *message.Message

		select {
		case roomBookingConfirmedChMsg = <-roomBookingConfirmedCh:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for RoomBookingConfirmed event")
		}

		traceCtx := propagation.TraceContext{}.Extract(
			context.Background(),
			propagation.MapCarrier(roomBookingConfirmedChMsg.Metadata),
		)
		sc := trace.SpanContextFromContext(traceCtx)

		require.True(
			t,
			sc.IsValid(),
			"RoomBookingConfirmed has no trace information (should be stored in 'traceparent' metadata): %v. "+
				"Did you added the middleware?",
			roomBookingConfirmedChMsg.Metadata,
		)

		assert.Equal(
			t,
			span.SpanContext().TraceID().String(),
			sc.TraceID().String(),
			"RoomBookingConfirmed event should share trace ID with PaymentReceived event",
		)
	}

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1)
}
