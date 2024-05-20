package rds

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
)

func NewSubscriber(redisClient *redis.Client, consumerGroup string, logger watermill.LoggerAdapter) (*redisstream.Subscriber, error) {
	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        redisClient,
		ConsumerGroup: consumerGroup,
	}, logger)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return subscriber, nil
}
