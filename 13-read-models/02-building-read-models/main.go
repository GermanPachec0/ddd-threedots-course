package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/shopspring/decimal"
)

type InvoiceIssued struct {
	InvoiceID    string
	CustomerName string
	Amount       decimal.Decimal
	IssuedAt     time.Time
}

type InvoicePaymentReceived struct {
	PaymentID  string
	InvoiceID  string
	PaidAmount decimal.Decimal
	PaidAt     time.Time

	FullyPaid bool
}

type InvoiceVoided struct {
	InvoiceID string
	VoidedAt  time.Time
}

type InvoiceReadModel struct {
	InvoiceID    string
	CustomerName string
	Amount       decimal.Decimal
	IssuedAt     time.Time

	FullyPaid     bool
	PaidAmount    decimal.Decimal
	LastPaymentAt time.Time

	Voided   bool
	VoidedAt time.Time
}

type InvoiceReadModelStorage struct {
	invoices      map[string]InvoiceReadModel
	payedInvoices map[string]decimal.Decimal
}

func NewInvoiceReadModelStorage() *InvoiceReadModelStorage {
	return &InvoiceReadModelStorage{
		invoices:      make(map[string]InvoiceReadModel),
		payedInvoices: make(map[string]decimal.Decimal),
	}
}

func (s *InvoiceReadModelStorage) Invoices() []InvoiceReadModel {
	invoices := make([]InvoiceReadModel, 0, len(s.invoices))
	for _, invoice := range s.invoices {
		invoices = append(invoices, invoice)
	}
	return invoices
}

func (s *InvoiceReadModelStorage) InvoiceByID(id string) (InvoiceReadModel, bool) {
	invoice, ok := s.invoices[id]
	return invoice, ok
}

func (s *InvoiceReadModelStorage) OnInvoiceIssued(ctx context.Context, event *InvoiceIssued) error {
	_, ok := s.InvoiceByID(event.InvoiceID)
	if ok {
		return nil
	}
	var invoice InvoiceReadModel
	invoice.InvoiceID = event.InvoiceID
	invoice.CustomerName = event.CustomerName
	invoice.Amount = event.Amount
	invoice.IssuedAt = event.IssuedAt

	s.invoices[event.InvoiceID] = invoice
	return nil
}

func (s *InvoiceReadModelStorage) OnInvoicePaymentReceived(ctx context.Context, event *InvoicePaymentReceived) error {
	invoice, ok := s.InvoiceByID(event.InvoiceID)
	if !ok {
		return fmt.Errorf("invoice %s not found", event.InvoiceID)
	}
	_, ok = s.payedInvoices[event.PaymentID]
	if ok {
		return nil
	}

	s.payedInvoices[event.PaymentID] = event.PaidAmount

	invoice.FullyPaid = event.FullyPaid
	invoice.PaidAmount = invoice.PaidAmount.Add(event.PaidAmount)
	invoice.LastPaymentAt = event.PaidAt
	s.invoices[event.InvoiceID] = invoice
	return nil
}

func (s *InvoiceReadModelStorage) OnInvoiceVoided(ctx context.Context, event *InvoiceVoided) error {
	invoice, ok := s.invoices[event.InvoiceID]
	if !ok {
		return fmt.Errorf("invoice %s not found", event.InvoiceID)
	}

	invoice.Voided = true
	invoice.VoidedAt = event.VoidedAt

	s.invoices[event.InvoiceID] = invoice
	return nil
}

func NewRouter(storage *InvoiceReadModelStorage, eventProcessorConfig cqrs.EventProcessorConfig, watermillLogger watermill.LoggerAdapter) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("could not create router: %w", err)
	}

	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, eventProcessorConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create command processor: %w", err)
	}

	err = eventProcessor.AddHandlers(
		cqrs.NewEventHandler(
			"OnInvoiceIssued",
			storage.OnInvoiceIssued,
		),
		cqrs.NewEventHandler(
			"OnInvoicePaymentReceived",
			storage.OnInvoicePaymentReceived,
		),
		cqrs.NewEventHandler(
			"OnInvoiceVoided",
			storage.OnInvoiceVoided,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not add event handlers: %w", err)
	}

	return router, nil
}
