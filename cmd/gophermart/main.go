package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env"
	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/infra"
	"github.com/devsagul/gophemart/internal/storage"
)

const ORDERS_BUFFER_SIZE = 255

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
	// todo add accrual poller
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Could not parse config for environment: %v", err)
	}

	flag.Parse()

	log.Println("Initializing storage...")
	// todo goroutine to ping database
	var store storage.Storage

	if cfg.DatabaseDsn == "" {
		store = storage.NewMemStorage()
	} else {
		store, err = storage.NewPostgresStorage(cfg.DatabaseDsn)
		if err != nil {
			log.Fatalf("Could not initialize postgres database: %v", err)
		}
	}

	log.Println("Initializing application...")
	accrualStream := make(chan *core.Order, ORDERS_BUFFER_SIZE)

	go func() {
		t := time.NewTicker(time.Minute)
		for range t.C {
			log.Print("Tick")
			orders, err := store.ExtractUnterminatedOrders()
			if err != nil {
				log.Printf("Error while extracting unterminated orders: %v", err)
				continue
			}

			log.Printf("Collected %d orders", len(orders))

			for _, order := range orders {

				log.Printf("Adding order to process: %s", order.Id)
				accrualStream <- order
			}
		}
	}()

	if cfg.AccrualAddress != "" {
		go infra.Worker(accrualStream, cfg.AccrualAddress, store)
	}

	app := infra.NewApp(store, accrualStream)
	err = app.HydrateKeys()
	// todo goroutine for keys hydration
	if err != nil {
		log.Fatalf("Could not hydrate the keys: %v", err)
	}

	err = http.ListenAndServe(cfg.Address, app.Router)
	if err != nil {
		log.Fatalf("Could not start the HTTP server: %v", err)
	}
}
