package sagas

import (
	"context"
	"fmt"
	"tickets/entities"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
)

type VipBundle struct {
	VipBundleID uuid.UUID `json:"vip_bundle_id"`

	BookingID       uuid.UUID  `json:"booking_id"`
	CustomerEmail   string     `json:"customer_email"`
	NumberOfTickets int        `json:"number_of_tickets"`
	ShowId          uuid.UUID  `json:"show_id"`
	BookingMadeAt   *time.Time `json:"booking_made_at"`

	TicketIDs []uuid.UUID `json:"ticket_ids"`

	Passengers []string `json:"passengers"`

	InboundFlightID         uuid.UUID   `json:"inbound_flight_id"`
	InboundFlightBookedAt   *time.Time  `json:"inbound_flight_booked_at"`
	InboundFlightTicketsIDs []uuid.UUID `json:"inbound_flight_tickets_ids"`

	ReturnFlightID         uuid.UUID   `json:"return_flight_id"`
	ReturnFlightBookedAt   *time.Time  `json:"return_flight_booked_at"`
	ReturnFlightTicketsIDs []uuid.UUID `json:"return_flight_tickets_ids"`

	TaxiBookedAt  *time.Time `json:"taxi_booked_at"`
	TaxiBookingID *uuid.UUID `json:"taxi_booking_id"`

	IsFinalized bool `json:"finalized"`
	Failed      bool `json:"failed"`
}

func NewVipBundle(
	vipBundleID uuid.UUID,
	bookingID uuid.UUID,
	customerEmail string,
	numberOfTickets int,
	showId uuid.UUID,
	passengers []string,
	inboundFlightID uuid.UUID,
	returnFlightID uuid.UUID,
) (*VipBundle, error) {
	if vipBundleID == uuid.Nil {
		return nil, fmt.Errorf("vip bundle id must be set")
	}
	if bookingID == uuid.Nil {
		return nil, fmt.Errorf("booking id must be set")
	}
	if customerEmail == "" {
		return nil, fmt.Errorf("customer email must be set")
	}
	if numberOfTickets <= 0 {
		return nil, fmt.Errorf("number of tickets must be greater than 0")
	}
	if showId == uuid.Nil {
		return nil, fmt.Errorf("show id must be set")
	}
	if numberOfTickets != len(passengers) {
		return nil, fmt.Errorf("number of tickets and passengers count mismatch")
	}
	if inboundFlightID == uuid.Nil {
		return nil, fmt.Errorf("inbound flight id must be set")
	}
	if returnFlightID == uuid.Nil {
		return nil, fmt.Errorf("return flight id must be set")
	}

	return &VipBundle{
		VipBundleID:     vipBundleID,
		BookingID:       bookingID,
		CustomerEmail:   customerEmail,
		NumberOfTickets: numberOfTickets,
		ShowId:          showId,
		Passengers:      passengers,
		InboundFlightID: inboundFlightID,
		ReturnFlightID:  returnFlightID,
	}, nil
}

type VipBundleRepository interface {
	Add(ctx context.Context, vipBundle VipBundle) error
	Get(ctx context.Context, vipBundleID uuid.UUID) (VipBundle, error)
	GetByBookingID(ctx context.Context, bookingID uuid.UUID) (VipBundle, error)

	UpdateByID(
		ctx context.Context,
		bookingID uuid.UUID,
		updateFn func(vipBundle VipBundle) (VipBundle, error),
	) (VipBundle, error)

	UpdateByBookingID(
		ctx context.Context,
		bookingID uuid.UUID,
		updateFn func(vipBundle VipBundle) (VipBundle, error),
	) (VipBundle, error)
}

type VipBundleProcessManager struct {
	commandBus *cqrs.CommandBus
	eventBus   *cqrs.EventBus
	repository VipBundleRepository
}

func NewVipBundleProcessManager(
	commandBus *cqrs.CommandBus,
	eventBus *cqrs.EventBus,
	repository VipBundleRepository,
) *VipBundleProcessManager {
	return &VipBundleProcessManager{
		commandBus: commandBus,
		eventBus:   eventBus,
		repository: repository,
	}
}

func (v VipBundleProcessManager) OnVipBundleInitialized(ctx context.Context, event *entities.VipBundleInitialized_v1) error {
	vb, err := v.repository.Get(ctx, event.VipBundleID)
	if err != nil {
		return err
	}

	return v.commandBus.Send(ctx, entities.BookShowTickets{
		BookingID:       vb.BookingID,
		CustomerEmail:   vb.CustomerEmail,
		NumberOfTickets: vb.NumberOfTickets,
		ShowId:          vb.ShowId,
	})
}

func (v VipBundleProcessManager) OnBookingMade(ctx context.Context, event *entities.BookingMade_v1) error {
	vb, err := v.repository.UpdateByBookingID(
		ctx,
		event.BookingID,
		func(vipBundle VipBundle) (VipBundle, error) {
			vipBundle.BookingMadeAt = &event.Header.PublishedAt
			return vipBundle, nil
		},
	)
	if err != nil {
		return err
	}

	return v.commandBus.Send(ctx, entities.BookFlight{
		CustomerEmail:  vb.CustomerEmail,
		FlightID:       vb.InboundFlightID,
		Passengers:     vb.Passengers,
		ReferenceID:    vb.VipBundleID.String(),
		IdempotencyKey: uuid.NewString(),
	})
}

func (v VipBundleProcessManager) OnTicketBookingConfirmed(ctx context.Context, event *entities.TicketBookingConfirmed_v1) error {
	_, err := v.repository.UpdateByBookingID(
		ctx,
		uuid.MustParse(event.BookingID),
		func(vipBundle VipBundle) (VipBundle, error) {
			eventTicketID := uuid.MustParse(event.TicketID)

			for _, ticketID := range vipBundle.TicketIDs {
				if ticketID == eventTicketID {
					continue
				}
			}

			vipBundle.TicketIDs = append(vipBundle.TicketIDs, eventTicketID)

			return vipBundle, nil
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (v VipBundleProcessManager) OnBookingFailed(ctx context.Context, event *entities.BookingFailed_v1) error {
	vb, err := v.repository.GetByBookingID(ctx, event.BookingID)
	if err != nil {
		return err
	}

	return v.rollbackProcess(ctx, vb.VipBundleID)
}

func (v VipBundleProcessManager) OnFlightBooked(ctx context.Context, event *entities.FlightBooked_v1) error {
	vb, err := v.repository.UpdateByID(
		ctx,
		uuid.MustParse(event.ReferenceID),
		func(vipBundle VipBundle) (VipBundle, error) {
			if vipBundle.InboundFlightID == event.FlightID {
				vipBundle.InboundFlightBookedAt = &event.Header.PublishedAt
				vipBundle.InboundFlightTicketsIDs = event.TicketIDs
			}
			if vipBundle.ReturnFlightID == event.FlightID {
				vipBundle.ReturnFlightBookedAt = &event.Header.PublishedAt
				vipBundle.ReturnFlightTicketsIDs = event.TicketIDs
			}

			return vipBundle, nil
		},
	)
	if err != nil {
		return err
	}

	switch {
	case vb.InboundFlightBookedAt != nil && vb.ReturnFlightBookedAt == nil:
		return v.commandBus.Send(ctx, entities.BookFlight{
			CustomerEmail:  vb.CustomerEmail,
			FlightID:       vb.ReturnFlightID,
			Passengers:     vb.Passengers,
			ReferenceID:    vb.VipBundleID.String(),
			IdempotencyKey: uuid.NewString(),
		})
	case vb.InboundFlightBookedAt != nil && vb.ReturnFlightBookedAt != nil:
		return v.commandBus.Send(ctx, entities.BookTaxi{
			CustomerEmail:      vb.CustomerEmail,
			CustomerName:       vb.Passengers[0],
			NumberOfPassengers: vb.NumberOfTickets,
			ReferenceID:        vb.VipBundleID.String(),
			IdempotencyKey:     uuid.NewString(),
		})
	default:
		return fmt.Errorf(
			"unsupported state: InboundFlightBookedAt: %v, ReturnFlightBookedAt: %v",
			vb.InboundFlightBookedAt,
			vb.ReturnFlightBookedAt,
		)
	}
}

func (v VipBundleProcessManager) OnFlightBookingFailed(ctx context.Context, event *entities.FlightBookingFailed_v1) error {
	return v.rollbackProcess(ctx, uuid.MustParse(event.ReferenceID))
}

func (v VipBundleProcessManager) OnTaxiBooked(ctx context.Context, event *entities.TaxiBooked_v1) error {
	vb, err := v.repository.UpdateByID(
		ctx,
		uuid.MustParse(event.ReferenceID),
		func(vb VipBundle) (VipBundle, error) {
			vb.TaxiBookedAt = &event.Header.PublishedAt
			vb.TaxiBookingID = &event.TaxiBookingID

			vb.IsFinalized = true

			return vb, nil
		},
	)
	if err != nil {
		return err
	}

	return v.eventBus.Publish(ctx, entities.VipBundleFinalized_v1{
		Header:      entities.NewEventHeader(),
		VipBundleID: vb.VipBundleID,
	})
}

func (v VipBundleProcessManager) OnTaxiBookingFailed(ctx context.Context, event *entities.TaxiBookingFailed_v1) error {
	return v.rollbackProcess(ctx, uuid.MustParse(event.ReferenceID))
}

func (v VipBundleProcessManager) rollbackProcess(ctx context.Context, vipBundleID uuid.UUID) error {
	vb, err := v.repository.Get(ctx, vipBundleID)
	if err != nil {
		return err
	}

	if vb.BookingMadeAt != nil {
		if err := v.rollbackTickets(ctx, vb); err != nil {
			return err
		}
	}
	if vb.InboundFlightBookedAt != nil {
		if err := v.commandBus.Send(ctx, entities.CancelFlightTickets{
			FlightTicketIDs: vb.InboundFlightTicketsIDs,
		}); err != nil {
			return err
		}
	}
	if vb.ReturnFlightBookedAt != nil {
		if err := v.commandBus.Send(ctx, entities.CancelFlightTickets{
			FlightTicketIDs: vb.ReturnFlightTicketsIDs,
		}); err != nil {
			return err
		}
	}

	_, err = v.repository.UpdateByID(
		ctx,
		vb.VipBundleID,
		func(vb VipBundle) (VipBundle, error) {
			vb.IsFinalized = true
			vb.Failed = true
			return vb, nil
		},
	)

	return err
}

func (v VipBundleProcessManager) rollbackTickets(ctx context.Context, vb VipBundle) error {
	if len(vb.TicketIDs) != vb.NumberOfTickets {
		return fmt.Errorf(
			"invalid number of tickets, expected %d, has %d: not all of TicketBookingConfirmed_v1 events were processed",
			vb.NumberOfTickets,
			len(vb.TicketIDs),
		)
	}

	for _, ticketID := range vb.TicketIDs {
		if err := v.commandBus.Send(ctx, entities.RefundTicket{
			Header:   entities.NewEventHeader(),
			TicketID: ticketID.String(),
		}); err != nil {
			return err
		}
	}

	return nil
}
