package rds

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
)

func NewRedisPublisher(redisClient *redis.Client, logger watermill.LoggerAdapter) (*redisstream.Publisher, error) {
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, logger)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return publisher, nil
}
