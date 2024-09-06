package rlmd

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// A user is allowed to make a maximum of requests per second
	maxRequestsPerSecond = 2
	dbMaxRetries         = 5
)

type RateLimiter struct {
	client *redis.Client
	logger *zap.Logger
}

func secretToKey(secret string) string {
	return "user:sec:" + secret
}

func secretToCountKey(secret string) string {
	return "user:sec:count:" + secret
}

func (s *RateLimiter) CheckSecret(secret string) (int, error) {
	val, err := s.client.Get(context.Background(), secretToKey(secret)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return http.StatusForbidden, err
		}
		return http.StatusInternalServerError, err
	}
	if val != secret {
		return http.StatusForbidden, errors.New("invalid secret")
	}
	return http.StatusOK, nil
}

func (s *RateLimiter) Allow(ctx context.Context, req *http.Request) (status int, err error) {
	var secret string
	if req.URL.Query().Has("secret") {
		secret = req.URL.Query().Get("secret")
	} else {
		secret = req.Header.Get("Demo-Secret")
	}
	if secret == "" {
		return http.StatusBadRequest, errors.New("empty secret")
	}
	status, err = s.CheckSecret(secret)
	if status != http.StatusOK {
		return status, err
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
		if count >= maxRequestsPerSecond {
			// No more requests associated with the current secret
			// are allowed in the current second
			s.logger.Debug("requests per second exceeded", zap.String("user secret", secret))
			status = http.StatusTooManyRequests
			return errors.New("too many requests")
		}
		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			if count > 0 {
				p.Incr(ctx, key)
			} else {
				p.Set(ctx, key, count+1, time.Second)
			}
			status = http.StatusOK
			return nil
		})

		return err
	}
	for i := 0; i < dbMaxRetries; i++ {
		err := s.client.Watch(ctx, txf, key)
		if err == nil {
			return status, err
		}
		if err == redis.TxFailedErr {
			continue
		}
		return status, err
	}
	return http.StatusInternalServerError, errors.New("transaction reached maximum number of retries")
}

func (s *RateLimiter) Close() {
	if err := s.client.Close(); err != nil {
		s.logger.Error("closing redis", zap.Error(err))
	}
}

func New(settings *Settings, logger *zap.Logger) (*RateLimiter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     settings.RedisAddress,
		Username: settings.RedisUser,
		Password: settings.RedisPassword,
		DB:       settings.RedisDB,
	})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*2))
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}
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
	}, nil
}

type Settings struct {
	RedisAddress  string
	RedisUser     string
	RedisPassword string
	RedisDB       int
	SecretsFile   string
}
