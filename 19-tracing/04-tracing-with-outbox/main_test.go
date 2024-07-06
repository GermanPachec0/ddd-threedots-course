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
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func Test(t *testing.T) {
	exp := &tracetest.InMemoryExporter{}

	tp := newTraceProvider(exp)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.TraceContext{})

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

	msgToPublish, publishSpan := publishMessage(t, err, db, watermillLogger)

	select {
	case <-time.After(time.Second * 5):
		t.Fatal("timeout waiting for message")
	case msg := <-messages:
		assert.Equal(t, msgToPublish.UUID, msg.UUID)
		assert.Equal(t, msgToPublish.Payload, msg.Payload)

		traceCtx := propagation.TraceContext{}.Extract(
			context.Background(),
			propagation.MapCarrier(msg.Metadata),
		)

		sc := trace.SpanContextFromContext(traceCtx)

		require.True(
			t,
			sc.IsValid(),
			"ItemAddedToCart has no trace information (should be stored in 'traceparent' metadata), metadata: %v",
			msg.Metadata,
		)

		t.Log(
			"ItemAddedToCart trace ID:", sc.TraceID().String(),
			"Metadata:", fmt.Sprintf("%v", msg.Metadata),
		)

		assert.Equal(
			t,
			publishSpan.SpanContext().TraceID().String(),
			sc.TraceID().String(),
			"PaymentReceived should maintain trace ID after publishing",
		)
	}
}

func publishMessage(t *testing.T, err error, db *sqlx.DB, watermillLogger watermill.LoggerAdapter) (*watermillMessage.Message, trace.Span) {
	ctx, span := otel.Tracer("").Start(context.Background(), "Publish")
	defer span.End()

	tx, err := db.Begin()
	require.NoError(t, err)

	msgToPublish := watermillMessage.NewMessage(uuid.NewString(), []byte("1234"))
	msgToPublish.SetContext(ctx)

	err = PublishInTx(msgToPublish, tx, watermillLogger)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	return msgToPublish, span
}

func newTraceProvider(exp *tracetest.InMemoryExporter) *sdktrace.TracerProvider {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("ExampleService"),
		),
	)
	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithResource(r),
	)
}
