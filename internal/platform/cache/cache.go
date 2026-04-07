package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/shukisha/backend/internal/platform/config"
)

var RDB *redis.Client
var ctx = context.Background()

// Init initializes the Redis connection using environment variables
func Init() (*redis.Client, error) {
	if redisURL := config.GetEnv("REDIS_URL", ""); redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
		}
		RDB = redis.NewClient(opt)
	} else {
		addr := fmt.Sprintf("%s:%s",
			config.GetEnv("REDIS_HOST", "localhost"),
			config.GetEnv("REDIS_PORT", "6379"),
		)

		RDB = redis.NewClient(&redis.Options{
			Addr:     addr,
			Username: config.GetEnv("REDIS_USER", ""),
			Password: config.GetEnv("REDIS_PASS", ""),
			DB:       0, // default DB
		})
	}

	if err := RDB.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	fmt.Println("Redis connection established successfully")
	return RDB, nil
}
