// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

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

func TestTrace(t *testing.T) {
	exp := &tracetest.InMemoryExporter{}

	tp := newTraceProvider(exp)

	otel.SetTracerProvider(tp)

	userID := "dbcdeaa7-e3fd-43b3-961c-7e5dc8f9419a"
	err := AddUser(
		context.Background(),
		userID,
		"Mariusz Pudzianowski",
	)
	require.NoError(t, err)

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1)

	allSpans := lo.Map(spans, func(item tracetest.SpanStub, _ int) string {
		return item.Name
	})
	allSpansStr := strings.Join(allSpans, ", ")

	addUserSpan, ok := lo.Find(spans, func(item tracetest.SpanStub) bool {
		return item.Name == "AddUser"
	})
	require.True(t, ok, "AddUser span not found, all spans: %s", allSpansStr)

	findUserSpan, ok := lo.Find(spans, func(item tracetest.SpanStub) bool {
		return item.Name == "FindUser"
	})
	require.True(t, ok, "FindUser span not found, all spans: %s", allSpansStr)

	tip := "Did you passed and used the context from AddUser to FindUser?"

	assert.Equal(
		t,
		addUserSpan.SpanContext.TraceID(),
		findUserSpan.SpanContext.TraceID(),
		"AddUser and FindUser spans should have the same TraceID. "+tip,
	)

	assert.Equal(
		t,
		findUserSpan.Parent.SpanID(),
		addUserSpan.SpanContext.SpanID(),
		"AddUser span should be the parent of FindUser span. "+tip,
	)

	addUserAttributes := lo.Map(addUserSpan.Attributes, func(item attribute.KeyValue, _ int) string {
		return string(item.Key)
	})
	addUserAttributesStr := strings.Join(addUserAttributes, ", ")

	addUserUserIdAttribute, ok := lo.Find(addUserSpan.Attributes, func(item attribute.KeyValue) bool {
		return item.Key == "userID"
	})
	require.True(t, ok, "AddUser span should have userID attribute, all attributes: %s", addUserAttributesStr)

	assert.Equal(
		t,
		userID,
		addUserUserIdAttribute.Value.AsString(),
		"AddUser span should have userID attribute with correct value. ",
	)

	findUserAttributes := lo.Map(findUserSpan.Attributes, func(item attribute.KeyValue, _ int) string {
		return string(item.Key)
	})
	findUserAttributesStr := strings.Join(findUserAttributes, ", ")

	findUserUserIdAttribute, ok := lo.Find(findUserSpan.Attributes, func(item attribute.KeyValue) bool {
		return item.Key == "userID"
	})

	require.True(t, ok, "FindUser span should have userID attribute, all attributes: %s", findUserAttributesStr)

	assert.Equal(
		t,
		userID,
		findUserUserIdAttribute.Value.AsString(),
		"FindUser span should have userID attribute with correct value.",
	)

	tipStatus := "Did you call span.SetStatus(codes.Error, err.Error()) when error is not nil?"

	assert.Equal(
		t,
		codes.Error,
		findUserSpan.Status.Code,
		"FindUser span should have error status. "+tipStatus,
	)
	assert.Equal(
		t,
		ErrUserNotFound.Error(),
		findUserSpan.Status.Description,
		"FindUser span should have error description. "+tipStatus,
	)
}
