package message

import (
	"fmt"
	"tickets/db"
	"tickets/entities"
	"tickets/message/command"
	"tickets/message/event"
	"tickets/message/outbox"
	"tickets/message/sagas"

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
	watermillLogger watermill.LoggerAdapter,
	vipBundleProcessManager *sagas.VipBundleProcessManager,
) *message.Router {
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

	err = cmdProccessor.AddHandlers(
		cqrs.NewCommandHandler(
			"TicketRefund",
			commandHandler.RefundTicket,
		),

		cqrs.NewCommandHandler(
			"BookShowTickets",
			commandHandler.BookShowTickets,
		),
		cqrs.NewCommandHandler(
			"BookFlight",
			commandHandler.BookFlight,
		),
		cqrs.NewCommandHandler(
			"BookTaxi",
			commandHandler.BookTaxi,
		),
		cqrs.NewCommandHandler(
			"CancelFlightTickets",
			commandHandler.CancelFlightTickets,
		),
	)
	if err != nil {
		panic(err)
	}

	err = eventProcessor.AddHandlers(
		cqrs.NewEventHandler(
			"BookPlaceInDeadNation",
			eventHandler.BookTicketToDeadNotion,
		),
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
			"PrintTicketHandler",
			eventHandler.StoreTicketsInFile,
		),
		cqrs.NewEventHandler(
			"StoreTickets",
			eventHandler.StoreTickets,
		),
		cqrs.NewEventHandler(
			"RemoveCanceledTicket",
			eventHandler.DeleteTicketCancel,
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
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnVipBundleInitialized",
			vipBundleProcessManager.OnVipBundleInitialized,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnBookingMade",
			vipBundleProcessManager.OnBookingMade,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnTicketBookingConfirmed",
			vipBundleProcessManager.OnTicketBookingConfirmed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnBookingFailed",
			vipBundleProcessManager.OnBookingFailed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnFlightBooked",
			vipBundleProcessManager.OnFlightBooked,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnFlightBookingFailed",
			vipBundleProcessManager.OnFlightBookingFailed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnTaxiBooked",
			vipBundleProcessManager.OnTaxiBooked,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnTaxiBookingFailed",
			vipBundleProcessManager.OnTaxiBookingFailed,
		),
	)

	if err != nil {
		panic(err)
	}

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
			event.EventPayload = msg.Payload
			event.EventID = event.Header.ID

			return dataLake.Create(msg.Context(), event)
		},
	)
	return router
}
