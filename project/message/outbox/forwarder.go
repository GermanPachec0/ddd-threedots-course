package outbox

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewForwarder(
	pgSusbscriber message.Subscriber,
	redisPub message.Publisher,
	logger watermill.LoggerAdapter,
	router *message.Router) (*forwarder.Forwarder, error) {

	fwd, err := forwarder.NewForwarder(pgSusbscriber, redisPub, logger,
		forwarder.Config{
			ForwarderTopic: topic,
			Router:         router,
			Middlewares: []message.HandlerMiddleware{
				func(h message.HandlerFunc) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						log.FromContext(msg.Context()).WithFields(logrus.Fields{
							"message_id": msg.UUID,
							"payload":    string(msg.Payload),
							"metadata":   msg.Metadata,
						}).Info("Forwarding message")
						return h(msg)
					}
				},
			},
		})
	if err != nil {
		return nil, err
	}

	return fwd, nil

}
