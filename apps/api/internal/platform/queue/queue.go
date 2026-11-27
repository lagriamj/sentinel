package queue

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is the API's Redis connection for later job coordination.
// It does not publish or consume scan jobs.
type Client struct {
	redis *redis.Client
}

// Open connects to Redis and checks that the server answers.
func Open(ctx context.Context, redisURL string) (*Client, error) {
	parsed, err := url.Parse(redisURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "redis" && parsed.Scheme != "rediss") {
		return nil, errors.New("redis url is invalid")
	}

	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, errors.New("redis url is invalid")
	}
	options.DialTimeout = 5 * time.Second
	options.ReadTimeout = 5 * time.Second
	options.WriteTimeout = 5 * time.Second

	client := redis.NewClient(options)
	connection := &Client{redis: client}
	if err := connection.Ping(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return connection, nil
}

// Ping reports whether Redis answers.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.redis == nil {
		return errors.New("redis client unavailable")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := c.redis.Ping(pingCtx).Err(); err != nil {
		return errors.New("redis ping failed")
	}
	return nil
}

// Close releases the Redis connection.
func (c *Client) Close() error {
	if c == nil || c.redis == nil {
		return nil
	}
	return c.redis.Close()
}
