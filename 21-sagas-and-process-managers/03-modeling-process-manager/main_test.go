// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type InMemoryVipBundleRepository struct {
	vipBundles     map[uuid.UUID]VipBundle
	vipBundlesLock sync.RWMutex
}

func (i *InMemoryVipBundleRepository) Add(ctx context.Context, vipBundle VipBundle) error {
	i.vipBundlesLock.Lock()
	defer i.vipBundlesLock.Unlock()

	i.vipBundles[vipBundle.VipBundleID] = vipBundle
	return nil
}

func (i *InMemoryVipBundleRepository) Get(ctx context.Context, vipBundleID uuid.UUID) (VipBundle, error) {
	i.vipBundlesLock.RLock()
	defer i.vipBundlesLock.RUnlock()

	if vipBundle, ok := i.vipBundles[vipBundleID]; ok {
		return vipBundle, nil
	}

	return VipBundle{}, fmt.Errorf("vip bundle with id %s not found", vipBundleID)
}

func (i *InMemoryVipBundleRepository) GetByBookingID(ctx context.Context, bookingID uuid.UUID) (VipBundle, error) {
	i.vipBundlesLock.RLock()
	defer i.vipBundlesLock.RUnlock()

	for _, vipBundle := range i.vipBundles {
		if vipBundle.BookingID == bookingID {
			return vipBundle, nil
		}
	}

	return VipBundle{}, fmt.Errorf("vip bundle with booking id %s not found", bookingID)
}

func (i *InMemoryVipBundleRepository) UpdateByID(
	ctx context.Context,
	bookingID uuid.UUID,
	updateFn func(vipBundle VipBundle) (VipBundle, error),
) (VipBundle, error) {
	i.vipBundlesLock.Lock()
	defer i.vipBundlesLock.Unlock()

	vipBundle, ok := i.vipBundles[bookingID]
	if !ok {
		return VipBundle{}, fmt.Errorf("vip bundle with id %s not found", bookingID)
	}

	vipBundle, err := updateFn(vipBundle)
	if err != nil {
		return VipBundle{}, err
	}

	i.vipBundles[vipBundle.VipBundleID] = vipBundle

	return vipBundle, nil
}

func (i *InMemoryVipBundleRepository) UpdateByBookingID(
	ctx context.Context,
	bookingID uuid.UUID,
	updateFn func(vipBundle VipBundle) (VipBundle, error),
) (VipBundle, error) {
	i.vipBundlesLock.Lock()
	defer i.vipBundlesLock.Unlock()

	for _, vipBundle := range i.vipBundles {
		if vipBundle.BookingID == bookingID {
			vipBundle, err := updateFn(vipBundle)
			if err != nil {
				return VipBundle{}, err
			}

			i.vipBundles[vipBundle.VipBundleID] = vipBundle

			return vipBundle, nil
		}
	}

	return VipBundle{}, fmt.Errorf("vip bundle with booking id %s not found", bookingID)
}

type InMemoryCommandBus struct {
	Commands []any
}

func (i *InMemoryCommandBus) Send(ctx context.Context, command any) error {
	i.Commands = append(i.Commands, command)
	return nil
}

func (i *InMemoryCommandBus) PopCommands() []any {
	commands := i.Commands
	i.Commands = nil
	return commands
}

type InMemoryEventBus struct {
	Events []any
}

func (i *InMemoryEventBus) Publish(ctx context.Context, event any) error {
	i.Events = append(i.Events, event)
	return nil
}

func (i *InMemoryEventBus) PopEvents() []any {
	events := i.Events
	i.Events = nil
	return events
}

func TestVipBundleProcessManager_successful_flow(t *testing.T) {
	commandBus := &InMemoryCommandBus{}
	eventBus := &InMemoryEventBus{}
	repo := &InMemoryVipBundleRepository{
		vipBundles: make(map[uuid.UUID]VipBundle),
	}

	pm := NewVipBundleProcessManager(commandBus, eventBus, repo)
	vb := newTestVipBundle(t)

	ctx := context.Background()

	err := repo.Add(ctx, *vb)
	require.NoError(t, err)

	err = pm.OnVipBundleInitialized(ctx, &VipBundleInitialized_v1{
		Header:      NewEventHeader(),
		VipBundleID: vb.VipBundleID,
	})
	require.NoError(t, err)

	commands := commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Equal(
		t,
		BookShowTickets{
			BookingID:       vb.BookingID,
			CustomerEmail:   vb.CustomerEmail,
			NumberOfTickets: vb.NumberOfTickets,
			ShowId:          vb.ShowId,
		},
		commands[0],
	)

	err = pm.OnBookingMade(ctx, &BookingMade_v1{
		Header:          NewEventHeader(),
		NumberOfTickets: vb.NumberOfTickets,
		BookingID:       vb.BookingID,
		CustomerEmail:   vb.CustomerEmail,
		ShowId:          vb.ShowId,
	})
	require.NoError(t, err)

	vbFromRepo, err := repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.BookingMadeAt)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookFlight{
				CustomerEmail: vb.CustomerEmail,
				FlightID:      vb.InboundFlightID,
				Passengers:    vb.Passengers,
				ReferenceID:   vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookFlight{}, "IdempotencyKey"),
		),
	)

	inboundFlightBooked := FlightBooked_v1{
		Header:      NewEventHeader(),
		FlightID:    vb.InboundFlightID,
		TicketIDs:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnFlightBooked(ctx, &inboundFlightBooked)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.InboundFlightBookedAt)
	assert.Equal(t, inboundFlightBooked.TicketIDs, vbFromRepo.InboundFlightTicketsIDs)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookFlight{
				CustomerEmail: vb.CustomerEmail,
				FlightID:      vb.ReturnFlightID,
				Passengers:    vb.Passengers,
				ReferenceID:   vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookFlight{}, "IdempotencyKey"),
		),
	)

	returnFlightBooked := FlightBooked_v1{
		Header:      NewEventHeader(),
		FlightID:    vb.ReturnFlightID,
		TicketIDs:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnFlightBooked(ctx, &returnFlightBooked)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.ReturnFlightBookedAt)
	assert.Equal(t, returnFlightBooked.TicketIDs, vbFromRepo.ReturnFlightTicketsIDs)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)

	assert.Empty(
		t,
		cmp.Diff(
			BookTaxi{
				CustomerEmail:      vb.CustomerEmail,
				CustomerName:       vb.Passengers[0],
				NumberOfPassengers: vb.NumberOfTickets,
				ReferenceID:        vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookTaxi{}, "IdempotencyKey"),
		),
	)

	taxiBooked := TaxiBooked_v1{
		Header:        NewEventHeader(),
		TaxiBookingID: uuid.New(),
		ReferenceID:   vb.VipBundleID.String(),
	}
	err = pm.OnTaxiBooked(ctx, &taxiBooked)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.TaxiBookedAt)
	assert.Equal(t, taxiBooked.TaxiBookingID, *vbFromRepo.TaxiBookingID)
	assert.True(t, vbFromRepo.IsFinalized)
	assert.False(t, vbFromRepo.Failed)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 0)

	events := eventBus.PopEvents()
	require.Len(t, events, 1)
	assert.Equal(
		t,
		vb.VipBundleID,
		events[0].(VipBundleFinalized_v1).VipBundleID,
	)
}

func TestVipBundleProcessManager_booking_failed(t *testing.T) {
	commandBus := &InMemoryCommandBus{}
	eventBus := &InMemoryEventBus{}
	repo := &InMemoryVipBundleRepository{
		vipBundles: make(map[uuid.UUID]VipBundle),
	}

	pm := NewVipBundleProcessManager(commandBus, eventBus, repo)
	vb := newTestVipBundle(t)

	ctx := context.Background()

	err := repo.Add(ctx, *vb)
	require.NoError(t, err)

	err = pm.OnVipBundleInitialized(ctx, &VipBundleInitialized_v1{
		Header:      NewEventHeader(),
		VipBundleID: vb.VipBundleID,
	})
	require.NoError(t, err)

	commands := commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Equal(
		t,
		BookShowTickets{
			BookingID:       vb.BookingID,
			CustomerEmail:   vb.CustomerEmail,
			NumberOfTickets: vb.NumberOfTickets,
			ShowId:          vb.ShowId,
		},
		commands[0],
	)

	err = pm.OnBookingFailed(ctx, &BookingFailed_v1{
		Header:    NewEventHeader(),
		BookingID: vb.BookingID,
	})
	require.NoError(t, err)

	vbFromRepo, err := repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.Nil(t, vbFromRepo.BookingMadeAt)
	assert.Nil(t, vbFromRepo.InboundFlightBookedAt)
	assert.True(t, vbFromRepo.IsFinalized)
	assert.True(t, vbFromRepo.Failed)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 0)

	events := eventBus.PopEvents()
	require.Len(t, events, 0)
}

func TestVipBundleProcessManager_inbound_flight_failed(t *testing.T) {
	commandBus := &InMemoryCommandBus{}
	eventBus := &InMemoryEventBus{}
	repo := &InMemoryVipBundleRepository{
		vipBundles: make(map[uuid.UUID]VipBundle),
	}

	pm := NewVipBundleProcessManager(commandBus, eventBus, repo)
	vb := newTestVipBundle(t)

	ctx := context.Background()

	err := repo.Add(ctx, *vb)
	require.NoError(t, err)

	err = pm.OnVipBundleInitialized(ctx, &VipBundleInitialized_v1{
		Header:      NewEventHeader(),
		VipBundleID: vb.VipBundleID,
	})
	require.NoError(t, err)

	commands := commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Equal(
		t,
		BookShowTickets{
			BookingID:       vb.BookingID,
			CustomerEmail:   vb.CustomerEmail,
			NumberOfTickets: vb.NumberOfTickets,
			ShowId:          vb.ShowId,
		},
		commands[0],
	)

	err = pm.OnBookingMade(ctx, &BookingMade_v1{
		Header:          NewEventHeader(),
		NumberOfTickets: vb.NumberOfTickets,
		BookingID:       vb.BookingID,
		CustomerEmail:   vb.CustomerEmail,
		ShowId:          vb.ShowId,
	})
	require.NoError(t, err)

	vbFromRepo, err := repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.BookingMadeAt)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookFlight{
				CustomerEmail: vb.CustomerEmail,
				FlightID:      vb.InboundFlightID,
				Passengers:    vb.Passengers,
				ReferenceID:   vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookFlight{}, "IdempotencyKey"),
		),
	)

	inboundFlightBookingFailed := FlightBookingFailed_v1{
		Header:      NewEventHeader(),
		FlightID:    vb.InboundFlightID,
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnFlightBookingFailed(ctx, &inboundFlightBookingFailed)
	require.Error(
		t,
		err,
		"it should fail, because TicketBookingConfirmed_v1 was not handled yet, "+
			"you should check if len(vipBundle.TicketIDs) == vipBundle.NumberOfTickets",
	)

	ticketIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, ticketID := range ticketIDs {
		ticketID := ticketID

		err = pm.OnTicketBookingConfirmed(ctx, &TicketBookingConfirmed_v1{
			Header:        NewEventHeader(),
			TicketID:      ticketID.String(),
			CustomerEmail: vb.CustomerEmail,
			Price: Money{
				Amount:   "100",
				Currency: "EUR",
			},
			BookingID: vb.BookingID.String(),
		})
		require.NoError(t, err)
	}

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	require.Len(t, vbFromRepo.TicketIDs, 3)

	err = pm.OnFlightBookingFailed(ctx, &inboundFlightBookingFailed)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.Nil(t, vbFromRepo.InboundFlightBookedAt)
	assert.True(t, vbFromRepo.IsFinalized)
	assert.True(t, vbFromRepo.Failed)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 3)

	assert.Equal(t, commands[0].(RefundTicket).TicketID, vbFromRepo.TicketIDs[0].String())
	assert.Equal(t, commands[1].(RefundTicket).TicketID, vbFromRepo.TicketIDs[1].String())
	assert.Equal(t, commands[2].(RefundTicket).TicketID, vbFromRepo.TicketIDs[2].String())

	events := eventBus.PopEvents()
	require.Len(t, events, 0)
}

func TestVipBundleProcessManager_return_flight_failed(t *testing.T) {
	commandBus := &InMemoryCommandBus{}
	eventBus := &InMemoryEventBus{}
	repo := &InMemoryVipBundleRepository{
		vipBundles: make(map[uuid.UUID]VipBundle),
	}

	pm := NewVipBundleProcessManager(commandBus, eventBus, repo)
	vb := newTestVipBundle(t)

	ctx := context.Background()

	err := repo.Add(ctx, *vb)
	require.NoError(t, err)

	err = pm.OnVipBundleInitialized(ctx, &VipBundleInitialized_v1{
		Header:      NewEventHeader(),
		VipBundleID: vb.VipBundleID,
	})
	require.NoError(t, err)

	commands := commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Equal(
		t,
		BookShowTickets{
			BookingID:       vb.BookingID,
			CustomerEmail:   vb.CustomerEmail,
			NumberOfTickets: vb.NumberOfTickets,
			ShowId:          vb.ShowId,
		},
		commands[0],
	)

	err = pm.OnBookingMade(ctx, &BookingMade_v1{
		Header:          NewEventHeader(),
		NumberOfTickets: vb.NumberOfTickets,
		BookingID:       vb.BookingID,
		CustomerEmail:   vb.CustomerEmail,
		ShowId:          vb.ShowId,
	})
	require.NoError(t, err)

	ticketIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, ticketID := range ticketIDs {
		ticketID := ticketID

		err = pm.OnTicketBookingConfirmed(ctx, &TicketBookingConfirmed_v1{
			Header:        NewEventHeader(),
			TicketID:      ticketID.String(),
			CustomerEmail: vb.CustomerEmail,
			Price: Money{
				Amount:   "100",
				Currency: "EUR",
			},
			BookingID: vb.BookingID.String(),
		})
		require.NoError(t, err)
	}

	vbFromRepo, err := repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.BookingMadeAt)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookFlight{
				CustomerEmail: vb.CustomerEmail,
				FlightID:      vb.InboundFlightID,
				Passengers:    vb.Passengers,
				ReferenceID:   vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookFlight{}, "IdempotencyKey"),
		),
	)

	inboundFlightBooked := FlightBooked_v1{
		Header:      NewEventHeader(),
		FlightID:    vb.InboundFlightID,
		TicketIDs:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnFlightBooked(ctx, &inboundFlightBooked)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.InboundFlightBookedAt)
	assert.Equal(t, inboundFlightBooked.TicketIDs, vbFromRepo.InboundFlightTicketsIDs)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookFlight{
				CustomerEmail: vb.CustomerEmail,
				FlightID:      vb.ReturnFlightID,
				Passengers:    vb.Passengers,
				ReferenceID:   vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookFlight{}, "IdempotencyKey"),
		),
	)

	returnFlightBooked := FlightBookingFailed_v1{
		Header:      NewEventHeader(),
		FlightID:    vb.ReturnFlightID,
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnFlightBookingFailed(ctx, &returnFlightBooked)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.Nil(t, vbFromRepo.ReturnFlightBookedAt)
	assert.True(t, vbFromRepo.IsFinalized)
	assert.True(t, vbFromRepo.Failed)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 4)

	assert.Equal(t, commands[0].(RefundTicket).TicketID, vbFromRepo.TicketIDs[0].String())
	assert.Equal(t, commands[1].(RefundTicket).TicketID, vbFromRepo.TicketIDs[1].String())
	assert.Equal(t, commands[2].(RefundTicket).TicketID, vbFromRepo.TicketIDs[2].String())

	assert.Equal(t, commands[3], CancelFlightTickets{
		FlightTicketIDs: vbFromRepo.InboundFlightTicketsIDs,
	})

	events := eventBus.PopEvents()
	require.Len(t, events, 0)
}

func TestVipBundleProcessManager_taxi_booking_failed(t *testing.T) {
	commandBus := &InMemoryCommandBus{}
	eventBus := &InMemoryEventBus{}
	repo := &InMemoryVipBundleRepository{
		vipBundles: make(map[uuid.UUID]VipBundle),
	}

	pm := NewVipBundleProcessManager(commandBus, eventBus, repo)
	vb := newTestVipBundle(t)

	ctx := context.Background()

	err := repo.Add(ctx, *vb)
	require.NoError(t, err)

	err = pm.OnVipBundleInitialized(ctx, &VipBundleInitialized_v1{
		Header:      NewEventHeader(),
		VipBundleID: vb.VipBundleID,
	})
	require.NoError(t, err)

	commands := commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Equal(
		t,
		BookShowTickets{
			BookingID:       vb.BookingID,
			CustomerEmail:   vb.CustomerEmail,
			NumberOfTickets: vb.NumberOfTickets,
			ShowId:          vb.ShowId,
		},
		commands[0],
	)

	err = pm.OnBookingMade(ctx, &BookingMade_v1{
		Header:          NewEventHeader(),
		NumberOfTickets: vb.NumberOfTickets,
		BookingID:       vb.BookingID,
		CustomerEmail:   vb.CustomerEmail,
		ShowId:          vb.ShowId,
	})
	require.NoError(t, err)

	vbFromRepo, err := repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.BookingMadeAt)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookFlight{
				CustomerEmail: vb.CustomerEmail,
				FlightID:      vb.InboundFlightID,
				Passengers:    vb.Passengers,
				ReferenceID:   vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookFlight{}, "IdempotencyKey"),
		),
	)

	inboundFlightBooked := FlightBooked_v1{
		Header:      NewEventHeader(),
		FlightID:    vb.InboundFlightID,
		TicketIDs:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnFlightBooked(ctx, &inboundFlightBooked)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.InboundFlightBookedAt)
	assert.Equal(t, inboundFlightBooked.TicketIDs, vbFromRepo.InboundFlightTicketsIDs)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookFlight{
				CustomerEmail: vb.CustomerEmail,
				FlightID:      vb.ReturnFlightID,
				Passengers:    vb.Passengers,
				ReferenceID:   vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookFlight{}, "IdempotencyKey"),
		),
	)

	returnFlightBooked := FlightBooked_v1{
		Header:      NewEventHeader(),
		FlightID:    vb.ReturnFlightID,
		TicketIDs:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnFlightBooked(ctx, &returnFlightBooked)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.NotNil(t, vbFromRepo.ReturnFlightBookedAt)
	assert.Equal(t, returnFlightBooked.TicketIDs, vbFromRepo.ReturnFlightTicketsIDs)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 1)
	assert.Empty(
		t,
		cmp.Diff(
			BookTaxi{
				CustomerEmail:      vb.CustomerEmail,
				CustomerName:       vb.Passengers[0],
				NumberOfPassengers: vb.NumberOfTickets,
				ReferenceID:        vb.VipBundleID.String(),
			},
			commands[0],
			cmpopts.IgnoreFields(BookTaxi{}, "IdempotencyKey"),
		),
	)

	ticketIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, ticketID := range ticketIDs {
		ticketID := ticketID

		err = pm.OnTicketBookingConfirmed(ctx, &TicketBookingConfirmed_v1{
			Header:        NewEventHeader(),
			TicketID:      ticketID.String(),
			CustomerEmail: vb.CustomerEmail,
			Price: Money{
				Amount:   "100",
				Currency: "EUR",
			},
			BookingID: vb.BookingID.String(),
		})
		require.NoError(t, err)
	}

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	require.Len(t, vbFromRepo.TicketIDs, 3)

	taxiBookingFailed := TaxiBookingFailed_v1{
		Header:      NewEventHeader(),
		ReferenceID: vb.VipBundleID.String(),
	}
	err = pm.OnTaxiBookingFailed(ctx, &taxiBookingFailed)
	require.NoError(t, err)

	vbFromRepo, err = repo.Get(ctx, vb.VipBundleID)
	require.NoError(t, err)
	assert.Nil(t, vbFromRepo.TaxiBookedAt)
	assert.Nil(t, vbFromRepo.TaxiBookingID)
	assert.True(t, vbFromRepo.IsFinalized)
	assert.True(t, vbFromRepo.Failed)

	commands = commandBus.PopCommands()
	require.Len(t, commands, 5)

	assert.Equal(t, commands[0].(RefundTicket).TicketID, vbFromRepo.TicketIDs[0].String())
	assert.Equal(t, commands[1].(RefundTicket).TicketID, vbFromRepo.TicketIDs[1].String())
	assert.Equal(t, commands[2].(RefundTicket).TicketID, vbFromRepo.TicketIDs[2].String())

	assert.Equal(t, commands[3], CancelFlightTickets{
		FlightTicketIDs: vbFromRepo.InboundFlightTicketsIDs,
	})
	assert.Equal(t, commands[4], CancelFlightTickets{
		FlightTicketIDs: vbFromRepo.ReturnFlightTicketsIDs,
	})

	events := eventBus.PopEvents()
	require.Len(t, events, 0)
}

func newTestVipBundle(t *testing.T) *VipBundle {
	vb, err := NewVipBundle(
		uuid.New(),
		uuid.New(),
		"example@example.com",
		3,
		uuid.New(),
		[]string{
			"Mariusz Pudzianowski",
			"Janusz Tracz",
			"Kamil Stoch",
		},
		uuid.New(),
		uuid.New(),
	)
	require.NoError(t, err)

	return vb
}
