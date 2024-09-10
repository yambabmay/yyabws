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
	dsRlmSettings *rlmd.Settings
}

const (
	// Default requests per second
	defaultRequestsPerSecond = 5
	// Maximum number of retries of a redis transaction
	dbMaxRetries = 5
	// Default location of the secrets file
	defaultSecretsFile = "./secrets.json"
)

// Get the settings from
func dsStreamRlmSettings() *rlmd.Settings {
	settings := &rlmd.Settings{
		RedisHost:         "localhost",
		RedisPort:         "6379",
		RedisMaxRetries:   dbMaxRetries,
		SecretsFile:       defaultSecretsFile,
		RequestsPerSecond: defaultRequestsPerSecond,
	}
	host := os.Getenv("REDIS_HOST")
	if host != "" {
		settings.RedisHost = host
	}
	port := os.Getenv("REDIS_PORT")
	if port != "" {
		settings.RedisPort = port
	}
	secrets := os.Getenv("USERS_SECRETS_FILE")
	if secrets != "" {
		settings.SecretsFile = secrets
	}
	user := os.Getenv("REDIS_USER")
	if user != "" {
		settings.RedisUser = user
	}
	password := os.Getenv("REDIS_PASSWORD")
	if password != "" {
		settings.RedisPassword = password
	}
	db := os.Getenv("REDIS_DB")
	if db != "" {
		val, err := strconv.Atoi(db)
		if err != nil {
			log.Fatal(fmt.Errorf("converting `REDIS_DB` value to int %v", err))
		}
		settings.RedisDB = val
	}
	rps := os.Getenv("DS_REQUESTS_PER_SECOND")
	if rps != "" {
		val, err := strconv.Atoi(rps)
		if err != nil {
			log.Fatal(fmt.Errorf("converting `DS_REQUESTS_PER_SECOND` value to int %v", err))
		}
		settings.RequestsPerSecond = val
	}
	mr := os.Getenv("REDIS_MAX_RETRIES")
	if mr != "" {
		val, err := strconv.Atoi(rps)
		if err != nil {
			log.Fatal(fmt.Errorf("converting `REDIS_MAX_RETRIES` value to int %v", err))
		}
		settings.RedisMaxRetries = val
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

	settings.dsRlmSettings = dsStreamRlmSettings()
	return settings
}
