package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/yambabmay/yyabws/server/rlmd"
)

type Settings struct {
	URL           string
	Secret        string
	MaxWaiting    int
	dsRlmSettings *rlmd.Settings
}

const (
	defaultMaxWaitingRequests = int(100)
)

func dsStreamRlmSettings() *rlmd.Settings {
	settings := &rlmd.Settings{}

	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	secrets := os.Getenv("USERS_SECRETS_FILE")
	if secrets == "" {
		secrets = "./secrets.json"
	}

	settings.RedisAddress = host + ":" + port
	settings.RedisUser = os.Getenv("REDIS_USER")
	settings.RedisPassword = os.Getenv("REDIS_PASSWORD")
	settings.SecretsFile = secrets

	db := os.Getenv("REDIS_DB")
	if db != "" {
		val, err := strconv.Atoi(db)
		if err != nil {
			log.Fatal(fmt.Errorf("converting `REDIS_DB` value to int %v", err))
		}
		settings.RedisDB = val
	}
	return settings
}

func loadSettings() *Settings {
	settings := &Settings{}

	secret := os.Getenv("ATLAS_SECRET")
	if secret == "" {
		log.Fatal("env variable ATLAS_SECRET is not set")
	}
	settings.Secret = secret

	url := os.Getenv("ATLAS_URL")
	if url == "" {
		url = "https://atlas.abiosgaming.com/v3"
	}
	settings.URL = url

	settings.MaxWaiting = defaultMaxWaitingRequests
	maxWaitingRequests := os.Getenv("MAX_WAITING_REQUESTS")
	if maxWaitingRequests != "" {
		val, err := strconv.Atoi(maxWaitingRequests)
		if err != nil {
			log.Fatal(fmt.Errorf("converting `MAX_WAITING_REQUESTS` value to int %v", err))
		}
		if val < 0 {
			log.Fatal(fmt.Errorf("invalid `MAX_WAITING_REQUESTS` value %v", val))
		}
		settings.MaxWaiting = val
	}

	settings.dsRlmSettings = dsStreamRlmSettings()
	return settings
}
