package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	transporthttp "github.com/banggok/sched_mind/backend/internal/transport/http"
)

func main() {
	address := os.Getenv("HTTP_ADDRESS")
	if address == "" {
		address = ":8080"
	}

	server := &http.Server{
		Addr:    address,
		Handler: transporthttp.NewRouter(),
	}

	log.Printf("API listening on %s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve API: %v", err)
	}
}
