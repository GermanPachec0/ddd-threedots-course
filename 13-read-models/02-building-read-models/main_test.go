// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(true, false)

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			BlockPublishUntilSubscriberAck: true,
		},
		watermillLogger,
	)
	storage := NewInvoiceReadModelStorage()

	eventBus, err := cqrs.NewEventBusWithConfig(
		pubSub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: cqrs.JSONMarshaler{
				GenerateName: cqrs.StructName,
			},
			Logger: watermillLogger,
		},
	)
	require.NoError(t, err)

	eventProcessorConfig := cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			return params.EventName, nil
		},
		SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return pubSub, nil
		},
		Marshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
		Logger: watermillLogger,
	}

	router, err := NewRouter(storage, eventProcessorConfig, watermillLogger)
	require.NoError(t, err)

	go func() {
		err := router.Run(context.Background())
		require.NoError(t, err)
	}()

	<-router.Running()

	invoiceIssuedEvents := []*InvoiceIssued{
		{
			InvoiceID:    uuid.NewString(),
			CustomerName: "Mariusz Pudzianowski",
			Amount:       decimal.NewFromInt(int64((rand.Intn(3) + 1) * 100)),
			IssuedAt:     time.Now().Truncate(time.Millisecond).UTC(),
		},
		{
			InvoiceID:    uuid.NewString(),
			CustomerName: "Janusz Tracz",
			Amount:       decimal.NewFromInt(int64((rand.Intn(3) + 1) * 100)),
			IssuedAt:     time.Now().Truncate(time.Millisecond).UTC(),
		},
	}

	for _, invoiceIssued := range invoiceIssuedEvents {
		err = eventBus.Publish(context.Background(), invoiceIssued)
		require.NoError(t, err)

		require.EventuallyWithT(
			t,
			func(t *assert.CollectT) {
				invoice, ok := storage.InvoiceByID(invoiceIssued.InvoiceID)
				if !assert.True(t, ok, "invoice %s not found", invoiceIssued.InvoiceID) {
					return
				}

				assert.Empty(
					t,
					cmp.Diff(
						InvoiceReadModel{
							InvoiceID:     invoiceIssued.InvoiceID,
							CustomerName:  invoiceIssued.CustomerName,
							Amount:        invoiceIssued.Amount,
							IssuedAt:      invoiceIssued.IssuedAt,
							FullyPaid:     false,
							PaidAmount:    decimal.Decimal{},
							LastPaymentAt: time.Time{},
							Voided:        false,
							VoidedAt:      time.Time{},
						},
						invoice,
					),
				)
			},
			time.Second,
			100*time.Millisecond,
			"read model should be updated",
		)
	}

	payInvoice(t, invoiceIssuedEvents[0], eventBus, storage)
	voidInvoice(t, invoiceIssuedEvents[1], eventBus, storage)
}

func voidInvoice(t *testing.T, issued *InvoiceIssued, bus *cqrs.EventBus, storage *InvoiceReadModelStorage) {
	invoiceVoidedEvent := &InvoiceVoided{
		InvoiceID: issued.InvoiceID,
		VoidedAt:  time.Now().Truncate(time.Millisecond).UTC(),
	}
	err := bus.Publish(context.Background(), invoiceVoidedEvent)
	require.NoError(t, err)

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			invoice, ok := storage.InvoiceByID(issued.InvoiceID)
			if !assert.True(t, ok, "invoice %s not found", issued.InvoiceID) {
				return
			}

			assert.Empty(
				t,
				cmp.Diff(
					InvoiceReadModel{
						InvoiceID:     issued.InvoiceID,
						CustomerName:  issued.CustomerName,
						Amount:        issued.Amount,
						IssuedAt:      issued.IssuedAt,
						FullyPaid:     false,
						PaidAmount:    decimal.Decimal{},
						LastPaymentAt: time.Time{},
						Voided:        true,
						VoidedAt:      invoiceVoidedEvent.VoidedAt,
					},
					invoice,
				),
			)
		},
		time.Second,
		100*time.Millisecond,
		"read model should be updated",
	)
}

func payInvoice(t *testing.T, invoiceIssued *InvoiceIssued, eventBus *cqrs.EventBus, storage *InvoiceReadModelStorage) {
	invoicePaymentReceivedEvent := &InvoicePaymentReceived{
		InvoiceID:  invoiceIssued.InvoiceID,
		PaymentID:  uuid.NewString(),
		PaidAmount: invoiceIssued.Amount.Mul(decimal.NewFromFloat(0.4)),
		PaidAt:     time.Now().Truncate(time.Millisecond).UTC(),
	}
	err := eventBus.Publish(context.Background(), invoicePaymentReceivedEvent)
	require.NoError(t, err)

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			invoice, ok := storage.InvoiceByID(invoiceIssued.InvoiceID)
			if !assert.True(t, ok, "invoice %s not found", invoiceIssued.InvoiceID) {
				return
			}

			assert.Empty(
				t,
				cmp.Diff(
					InvoiceReadModel{
						InvoiceID:     invoiceIssued.InvoiceID,
						CustomerName:  invoiceIssued.CustomerName,
						Amount:        invoiceIssued.Amount,
						IssuedAt:      invoiceIssued.IssuedAt,
						FullyPaid:     false,
						PaidAmount:    invoicePaymentReceivedEvent.PaidAmount,
						LastPaymentAt: invoicePaymentReceivedEvent.PaidAt,
						Voided:        false,
						VoidedAt:      time.Time{},
					},
					invoice,
				),
			)
		},
		time.Second,
		100*time.Millisecond,
		"read model should be updated",
	)

	leftover := invoiceIssued.Amount.Sub(invoicePaymentReceivedEvent.PaidAmount)

	invoicePaymentReceivedEvent = &InvoicePaymentReceived{
		InvoiceID:  invoiceIssued.InvoiceID,
		PaymentID:  uuid.NewString(),
		PaidAmount: leftover,
		PaidAt:     time.Now().Truncate(time.Millisecond).UTC(),
		FullyPaid:  true,
	}

	err = eventBus.Publish(context.Background(), invoicePaymentReceivedEvent)
	require.NoError(t, err)

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			invoice, ok := storage.InvoiceByID(invoiceIssued.InvoiceID)
			if !assert.True(t, ok, "invoice %s not found", invoiceIssued.InvoiceID) {
				return
			}

			assert.Empty(
				t,
				cmp.Diff(
					InvoiceReadModel{
						InvoiceID:     invoiceIssued.InvoiceID,
						CustomerName:  invoiceIssued.CustomerName,
						Amount:        invoiceIssued.Amount,
						IssuedAt:      invoiceIssued.IssuedAt,
						FullyPaid:     true,
						PaidAmount:    invoiceIssued.Amount, // this should be fully paid
						LastPaymentAt: invoicePaymentReceivedEvent.PaidAt,
						Voided:        false,
						VoidedAt:      time.Time{},
					},
					invoice,
				),
			)
		},
		time.Second,
		100*time.Millisecond,
		"read model should be updated",
	)
}
