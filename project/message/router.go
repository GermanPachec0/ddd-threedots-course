package message

import (
	"tickets/db"
	"tickets/message/command"
	"tickets/message/event"
	"tickets/message/outbox"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewWatermillRouter(
	pgSubscriber message.Subscriber,
	commandProccesorConfig cqrs.CommandProcessorConfig,
	publisher message.Publisher,
	eventProcessorConfig cqrs.EventProcessorConfig,
	commandHandler command.Handler,
	eventHandler event.Handler,
	opsReadModel db.OpsBookingReadModel,
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

	return router
}
