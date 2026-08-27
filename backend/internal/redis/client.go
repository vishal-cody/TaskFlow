// package redis provides redis connectivity for caching and rate limiting.
package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// newclient creates a redis client from a url string.
// the url should be in the form: redis://[:password@]host:port[/db]
func NewClient(ctx context.Context, url string, logger *slog.Logger) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parsing Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// verify connectivity.
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("pinging Redis: %w", err)
	}

	logger.Info("connected to Redis", "addr", opts.Addr, "db", opts.DB)
	return client, nil
}
