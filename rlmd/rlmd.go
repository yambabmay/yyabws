// This package manages the and acts on the downstream [client side]
// rate limiting information.
package rlmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Downstream rate limiter.
type RateLimiter struct {
	client *redis.Client
	logger *zap.Logger
	// Requests per second
	rps int
	// Max database retries
	mxr int
}

// secretToKey - make redis key of user secret
func secretToKey(secret string) string {
	return "user:sec:" + secret
}

// secretToCountKey - make redis key to identify the requests
// counter associated with an user secret
func secretToCountKey(secret string) string {
	return "user:sec:count:" + secret
}

// CheckSecret validate the secret.
func (s *RateLimiter) CheckSecret(secret string) (int, error) {
	val, err := s.client.Get(context.Background(), secretToKey(secret)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// The secret is not in the database
			return http.StatusForbidden, err
		}
		// Something else went wrong
		return http.StatusInternalServerError, err
	}
	if val != secret {
		return http.StatusForbidden, errors.New("invalid secret")
	}
	return http.StatusOK, nil
}

// Secret - Extract the secret from the request
func (s *RateLimiter) Secret(ctx context.Context, req *http.Request) (secret string, err error) {
	if req.URL.Query().Has("secret") {
		secret = req.URL.Query().Get("secret")
	} else {
		secret = req.Header.Get("Demo-Secret")
	}
	if secret == "" {
		return secret, errors.New("empty secret")
	}
	return secret, nil
}

// Allow - Check if the rate limiter allows a request
func (s *RateLimiter) Allow(ctx context.Context, req *http.Request) (status int, secret string, err error) {
	secret, err = s.Secret(ctx, req)
	if err != nil {
		return http.StatusBadRequest, secret, err
	}
	status, err = s.CheckSecret(secret)
	if status != http.StatusOK {
		return status, secret, err
	}
	key := secretToCountKey(secret)

	txf := func(tx *redis.Tx) error {
		// getting the current second request count
		count, err := tx.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			s.logger.Warn("getting the current second request count",
				zap.String("secret", secret),
				zap.Error(err))
			return err
		}
		if count >= s.rps {
			// Do nor allow requests associated with the
			// current secret in the current reset period
			s.logger.Debug("too many requests", zap.String("secret", secret))

			status = http.StatusTooManyRequests
			return ErrTooManyRequests
		}
		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			if count > 0 {
				// The count was in the database, increment it by one
				p.Incr(ctx, key)
			} else {
				// New bust, insert it in the database
				p.Set(ctx, key, count+1, time.Second)
			}
			status = http.StatusOK
			return nil
		})

		return err
	}
	for i := 0; i < s.mxr; i++ {
		err := s.client.Watch(ctx, txf, key)
		if err == nil {
			return status, secret, nil
		}
		if err == redis.TxFailedErr {
			continue
		}
		return status, secret, err
	}
	return http.StatusInternalServerError, secret, errors.New("transaction maximum retries")
}

// Info - Get the rate limiting flags associated with a secret to add to a response
func (s *RateLimiter) Info(ctx context.Context, secret string) (map[string]string, error) {
	m := make(map[string]string)
	if _, err := s.CheckSecret(secret); err != nil {
		return m, err
	}
	key := secretToCountKey(secret)
	txf := func(tx *redis.Tx) error {
		count, err := tx.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			s.logger.Warn("getting  remaining requests",
				zap.String("secret", secret),
				zap.Error(err))
			return err
		}
		remaining := s.mxr - count
		duration, err := tx.PTTL(ctx, key).Result()
		if err != nil && err != redis.Nil {
			s.logger.Warn("getting milliseconds to reset ",
				zap.String("secret", secret),
				zap.Error(err))
			return err
		}
		m["X-RateLimit-Limit"] = fmt.Sprintf("%d", s.rps)
		m["X-RateLimit-Burst"] = fmt.Sprintf("%d", s.rps)
		m["X-RateLimit-Remaining"] = fmt.Sprintf("%d", remaining)
		m["X-RateLimit-Reset"] = fmt.Sprintf("%d", duration.Milliseconds())
		m["Retry-After"] = fmt.Sprintf("%d", 0)
		if remaining == 0 {
			m["Retry-After"] = fmt.Sprintf("%d", time.Duration(1*time.Second).Milliseconds())
			m["X-RateLimit-Reset"] = fmt.Sprintf("%d", time.Duration(1*time.Second).Milliseconds())
		}
		return nil
	}
	for i := 0; i < s.mxr; i++ {
		err := s.client.Watch(ctx, txf, key)
		if err == nil {
			return m, nil
		}
		if err == redis.TxFailedErr {
			continue
		}
		return m, err
	}
	return m, errors.New("transaction maximum retries")
}

// Close - close the redis database
func (s *RateLimiter) Close() {
	if err := s.client.Close(); err != nil {
		s.logger.Error("closing redis", zap.Error(err))
	}
}

// New - Create a new Downstream rate limiter
func New(settings *Settings, logger *zap.Logger) (*RateLimiter, error) {
	// Create the redis client
	client := redis.NewClient(&redis.Options{
		Addr:     settings.RedisHost + ":" + settings.RedisPort,
		Username: settings.RedisUser,
		Password: settings.RedisPassword,
		DB:       settings.RedisDB,
	})
	// Check if the database is up
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*2))
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}
	// Read the secrets from a file
	data, err := os.ReadFile(settings.SecretsFile)
	if err != nil {
		log.Fatal("while opening secrets file: ", err)
	}
	var secrets []string
	err = json.Unmarshal(data, &secrets)
	if err != nil {
		log.Fatal("json Unmarshal secrets data: ", err)
	}
	// Add the secrets to the redis database
	for _, secret := range secrets {
		err := client.Set(context.Background(), secretToKey(secret), secret, 0).Err()
		if err != nil {
			log.Fatal(err)
		}
	}
	return &RateLimiter{
		client: client,
		logger: logger,
		rps:    settings.RequestsPerSecond,
		mxr:    settings.RedisMaxRetries,
	}, nil
}

type Settings struct {
	// RedisHost The redis host
	// Configurable through the environment
	// variable REDIS_HOST, defaults to "localhost"
	RedisHost string
	// RedisPort The redis port
	// Configurable through the environment
	// variable REDIS_PORT, defaults to "6379"
	RedisPort string
	// RedisUser The redis user
	// Configurable through the environment
	// variable REDIS_USER, defaults to ""
	RedisUser string
	// RedisPassword The redis password
	// Configurable through the environment
	// variable REDIS_PASSWORD, defaults to ""
	RedisPassword string
	// RedisDB The redis database
	// Configurable through the environment
	// variable REDIS_DB, defaults to 0
	RedisDB int
	// RedisMaxRetries The maximum number of transaction
	// Retries
	// Configurable through the environment
	// variable REDIS_MAX_RETRIES, defaults to 5
	RedisMaxRetries int
	// The source of secrets. We use a json file
	// that that parses to s slice of strings.
	// Configurable through the environment
	// variable USERS_SECRETS_FILE, defaults to "./secrets.json"
	// and can be recreated with the script in scripts/gen-secrets.sh
	SecretsFile string
	// Requests per second associated with a secret
	// Configurable through the environment
	// variable DS_REQUESTS_PER_SECOND, defaults to 5
	RequestsPerSecond int
}
