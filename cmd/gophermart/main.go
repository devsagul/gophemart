package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env"
	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/infra"
	"github.com/devsagul/gophemart/internal/storage"
)

const OrdersBufferSize = 255
const PollInterval = 30 * time.Second
const DatabaseHealthCheckInterval = time.Minute
const HidrationInterval = 12 * time.Hour

type config struct {
	Address        string `env:"RUN_ADDRESS"`
	DatabaseDsn    string `env:"DATABASE_URI"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

var cfg config

func init() {
	flag.StringVar(&cfg.Address, "a", "localhost:8000", "Address of the server (to listen to)")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "DSN to connect to the database (leave empty to use in-memory DB)")
	flag.StringVar(&cfg.AccrualAddress, "r", "", "Address of the accrual system")
}

func main() {
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Could not parse config for environment: %v", err)
	}

	flag.Parse()

	log.Println("Initializing storage...")
	var store storage.Storage

	go func() {
		t := time.NewTicker(PollInterval)
		for range t.C {
			err := store.Ping(context.Background())
			if err != nil {
				log.Printf("Error while checking health of the database: %v", err)
			}
		}
	}()

	if cfg.DatabaseDsn == "" {
		store = storage.NewMemStorage()
	} else {
		store, err = storage.NewPostgresStorage(cfg.DatabaseDsn)
		if err != nil {
			log.Fatalf("Could not initialize postgres database: %v", err)
		}
	}

	log.Println("Initializing application...")
	accrualStream := make(chan *core.Order, OrdersBufferSize)

	go func() {
		t := time.NewTicker(PollInterval)
		for range t.C {
			orders, err := store.ExtractUnterminatedOrders()
			if err != nil {
				log.Printf("Error while extracting unterminated orders: %v", err)
				continue
			}

			for _, order := range orders {
				accrualStream <- order
			}
		}
	}()

	if cfg.AccrualAddress != "" {
		go infra.Worker(accrualStream, cfg.AccrualAddress, store)
	}

	app := infra.NewApp(store, accrualStream)
	err = app.HydrateKeys()
	if err != nil {
		log.Fatalf("Could not hydrate the keys: %v", err)
	}

	go func() {
		t := time.NewTicker(PollInterval)
		for range t.C {
			err = app.HydrateKeys()
			if err != nil {
				log.Printf("Error while hydrating hmac keys: %v", err)
			}
		}
	}()

	err = http.ListenAndServe(cfg.Address, app.Router)
	if err != nil {
		log.Fatalf("Could not start the HTTP server: %v", err)
	}
}
