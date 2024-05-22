package main

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type FollowRequestSent struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type EventsCounter interface {
	CountEvent() error
}
type EventsHandler struct {
	counter EventsCounter
}

func (h *EventsHandler) Handle(ctx context.Context, event *FollowRequestSent) error {
	return h.counter.CountEvent()
}

func NewFollowRequestSentHandler(counter EventsCounter) cqrs.EventHandler {
	h := EventsHandler{
		counter: counter,
	}

	return cqrs.NewEventHandler(
		"FollowrequestSent",
		h.Handle,
	)
}
