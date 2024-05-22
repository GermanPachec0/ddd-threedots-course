package main

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/lithammer/shortuuid/v3"
)

type CorrelationPublisherDecorator struct {
	message.Publisher
}

func (c CorrelationPublisherDecorator) Publish(topic string, messages ...*message.Message) error {
	msg := messages[0]
	messageContext := msg.Context()
	msg.Metadata.Set("correlation_id", CorrelationIDFromContext(messageContext))

	return c.Publisher.Publish(topic, messages...)
}

type ctxKey int

const (
	correlationIDKey ctxKey = iota
)

func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

func CorrelationIDFromContext(ctx context.Context) string {
	v, ok := ctx.Value(correlationIDKey).(string)
	if ok {
		return v
	}

	// add "gen_" prefix to distinguish generated correlation IDs from correlation IDs passed by the client
	// it's useful to detect if correlation ID was not passed properly
	return "gen_" + shortuuid.New()
}
