package message

import (
	"fmt"
	"tickets/db"
	"tickets/entities"
	"tickets/message/command"
	"tickets/message/event"
	"tickets/message/outbox"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewWatermillRouter(
	pgSubscriber message.Subscriber,
	redisSub message.Subscriber,
	commandProccesorConfig cqrs.CommandProcessorConfig,
	publisher message.Publisher,
	eventProcessorConfig cqrs.EventProcessorConfig,
	commandHandler command.Handler,
	eventHandler event.Handler,
	opsReadModel db.OpsBookingReadModel,
	dataLake db.EventRepository,
	watermillLogger watermill.LoggerAdapter) *message.Router {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		panic(err)
	}

	useMiddlewares(router, watermillLogger)

	_, err = outbox.NewForwarder(pgSubscriber, publisher, watermillLogger, router)
	if err != nil {
		panic(err)
	}
	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, eventProcessorConfig)
	if err != nil {
		panic(err)
	}

	cmdProccessor, err := cqrs.NewCommandProcessorWithConfig(router, commandProccesorConfig)

	cmdProccessor.AddHandlers(
		cqrs.NewCommandHandler(
			"HandlefundTicket",
			commandHandler.RefundTicket,
		),
	)

	eventProcessor.AddHandlers(
		cqrs.NewEventHandler(
			"AppendToTracker",
			eventHandler.AppendToTracker,
		),
		cqrs.NewEventHandler(
			"TicketRefundToSheet",
			eventHandler.TicketRefundToSheet,
		),
		cqrs.NewEventHandler(
			"IssueReceipt",
			eventHandler.IssueReceipt,
		),
		cqrs.NewEventHandler(
			"SaveTicketInDB",
			eventHandler.StoreTickets,
		),
		cqrs.NewEventHandler(
			"DeleteCancelTicketsInDB",
			eventHandler.DeleteTicketCancel,
		),
		cqrs.NewEventHandler(
			"SaveTicketInFile",
			eventHandler.StoreTicketsInFile,
		),
		cqrs.NewEventHandler(
			"BookPlaceInDeadNation",
			eventHandler.BookTicketToDeadNotion,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnBookingMade",
			opsReadModel.OnBookingMade,
		),
		cqrs.NewEventHandler(
			"ops_read_model.IssueReceiptHandler",
			opsReadModel.OnTicketReceiptIssued,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketBookingConfirmed",
			opsReadModel.OnTicketBookingConfirmed,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketPrinted",
			opsReadModel.OnTicketPrinted,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketRefunded",
			opsReadModel.OnTicketRefunded,
		),
	)
	router.AddNoPublisherHandler(
		"events_splitter",
		"events",
		redisSub,
		func(msg *message.Message) error {
			eventName := eventProcessorConfig.Marshaler.NameFromMessage(msg)
			if eventName == "" {
				return fmt.Errorf("cannot get event name from message")
			}
			return publisher.Publish("events."+eventName, msg)
		},
	)

	router.AddNoPublisherHandler(
		"events_data_lake",
		"events",
		redisSub,
		func(msg *message.Message) error {
			var event entities.Event
			eventName := eventProcessorConfig.Marshaler.NameFromMessage(msg)
			if eventName == "" {
				return fmt.Errorf("cannot get event name from message")
			}
			if err := eventProcessorConfig.Marshaler.Unmarshal(msg, &event); err != nil {
				return fmt.Errorf("cannot unmarshal event: %w", err)
			}
			event.EventName = eventName
			event.EventPayload = string(msg.Payload)
			event.EventID = event.Header.ID
			return dataLake.Create(msg.Context(), event)
		},
	)
	return router
}
