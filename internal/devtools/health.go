package devtools

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
)

// WaitForPostgres polls until PostgreSQL accepts connections or timeout
func WaitForPostgres(ctx context.Context, host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("%s:%d", host, port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	return fmt.Errorf("timeout waiting for postgres at %s", addr)
}

// WaitForRedis polls until Redis responds to PING or timeout
func WaitForRedis(ctx context.Context, host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("%s:%d", host, port)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  1 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})
	defer client.Close()

	for time.Now().Before(deadline) {
		_, err := client.Ping(ctx).Result()
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	return fmt.Errorf("timeout waiting for redis at %s", addr)
}

// CheckPostgres checks if PostgreSQL is accessible
func CheckPostgres(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("postgres not accessible at %s: %w", addr, err)
	}
	conn.Close()
	return nil
}

// CheckRedis checks if Redis is accessible
func CheckRedis(ctx context.Context, host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	defer client.Close()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis not accessible at %s: %w", addr, err)
	}
	return nil
}
