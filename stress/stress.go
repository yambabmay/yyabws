package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

func sendRequest(endpoint, secret string, take, skip int, output bool, logger *zap.Logger) {
	bURL, _ := url.Parse(endpoint)
	values := url.Values{}
	values.Set("take", fmt.Sprintf("%d", take))
	values.Set("skip", fmt.Sprintf("%d", skip))
	bURL.RawQuery = values.Encode()
	req, _ := http.NewRequest(http.MethodGet, bURL.String(), nil)
	req.Header.Add("Demo-Secret", secret)
	cli := &http.Client{}
	rsp, err := cli.Do(req)
	if err != nil {
		fmt.Println("secret", err)
		return
	}
	defer rsp.Body.Close()
	logger.Debug("rate limits headers",
		zap.String("X-RateLimit-Limit", rsp.Header.Get("X-RateLimit-Limit")),
		zap.String("X-RateLimit-Burst", rsp.Header.Get("X-RateLimit-Burst")),
		zap.String("X-RateLimit-Remaining", rsp.Header.Get("X-RateLimit-Remaining")),
		zap.String("X-RateLimit-Reset", rsp.Header.Get("X-RateLimit-Reset")),
		zap.String("Retry-After", rsp.Header.Get("Retry-After")),
	)
	fmt.Println(secret, rsp.Status)
	if output {
		io.Copy(os.Stdout, rsp.Body)
		fmt.Println()
	}
}

func stress(
	endpoint string,
	secret string,
	count int,
	pause time.Duration,
	take int,
	skip int,
	wg *sync.WaitGroup,
	output bool,
	loger *zap.Logger,
) {
	defer wg.Done()
	for i := 0; i < count; i++ {
		sendRequest(endpoint, secret, take, skip, output, loger)
		time.Sleep(pause)
	}
}

func loadSecrets(secretsFile string) []string {
	// Read the secrets from a file

	data, err := os.ReadFile(secretsFile)
	if err != nil {
		log.Fatal("while opening secrets file: ", err)
	}
	var secrets []string
	err = json.Unmarshal(data, &secrets)
	if err != nil {
		log.Fatal("json Unmarshal secrets data: ", err)
	}
	return secrets
}

func main() {
	secretsCmd := flag.NewFlagSet("secrets", flag.ExitOnError)
	secretsCount := secretsCmd.Bool("count", true, "show the number of user secrets")
	secretsList := secretsCmd.Bool("list", false, "list the secrets")
	secretsFile := secretsCmd.String("file", "/usr/src/app/secrets.json", "users secrets file")

	stressCmd := flag.NewFlagSet("run", flag.ExitOnError)
	stressEndPoint := stressCmd.String("endpoint", "http://localhost:80/teams/live", "endpoint")
	stressFile := stressCmd.String("secrets", "/usr/src/app/secrets.json", "users secrets file")
	stressTake := stressCmd.Int("take", 1, "the number of records to take")
	stressSkip := stressCmd.Int("skip", 0, "the number of records to skip")
	stressPause := stressCmd.Duration("pause", time.Second, "request period")
	stressRounds := stressCmd.Int("rounds", 3, "number of rounds")
	stressClients := stressCmd.Int("clients", 5, "number of simulated clients")
	stressPrint := stressCmd.Bool("print", false, "print the responses")
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) < 2 {
		fmt.Println("expected 'run' or 'secrets' subcommands")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "run":
		stressCmd.Parse(os.Args[2:])
		secrets := loadSecrets(*stressFile)
		if *stressClients < 1 || *stressClients > len(secrets) {
			fmt.Printf("expect number of clients in the range [1 .. %d]\n", len(secrets))
			os.Exit(1)
		}
		var wg sync.WaitGroup
		for i := 0; i < *stressClients; i++ {
			wg.Add(1)
			go stress(
				*stressEndPoint,
				secrets[i],
				*stressRounds,
				*stressPause,
				*stressTake,
				*stressSkip,
				&wg,
				*stressPrint,
				logger,
			)
		}
		wg.Wait()

	case "secrets":
		secretsCmd.Parse(os.Args[2:])
		secrets := loadSecrets(*secretsFile)
		if *secretsCount {
			fmt.Println("Number of secrets: ", len(secrets))
		}

		if *secretsList {
			for _, secret := range secrets {
				fmt.Println(secret)
			}
		}
	default:
		fmt.Println("expected 'run' or 'secrets' subcommands")
		os.Exit(1)
	}
}
