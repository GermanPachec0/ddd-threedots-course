package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var database = map[string]string{
	"4710fec1-9e43-4d28-a0be-05113d383b79": "John Doe",
	"8e3f0d0d-f519-467d-904d-b0fd7461df8c": "Jane Doe",
	"947471cc-09ea-402e-9439-d7cd75a7a3a4": "Bob Builder",
}

var ErrUserNotFound = fmt.Errorf("user not found")

func FindUser(ctx context.Context, userID string) (string, error) {
	ctx, span := otel.Tracer("user").Start(ctx, "FindUser",
		trace.WithAttributes(
			attribute.String("userID", userID),
		))
	defer span.End()
	// Simulate a slow database read
	time.Sleep(time.Millisecond * 100)

	data, exists := database[userID]
	if !exists {
		return "", ErrUserNotFound
	}

	return data, nil
}

func AddUser(ctx context.Context, userID, name string) error {
	ctx, span := otel.Tracer("").Start(
		ctx,
		"AddUser",
		trace.WithAttributes(
			attribute.String("userID", userID),
		),
	)
	defer span.End()
	span.SetAttributes(
		attribute.String("userID", userID),
	)
	_, err := FindUser(ctx, userID)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to find user: %w", err)
	}
	span.SetAttributes(
		attribute.String("userID", userID),
	)
	// Simulate a slow database write
	time.Sleep(time.Millisecond * 150)
	database[userID] = name

	return nil
}
