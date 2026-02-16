package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/apgupta3091/job-runner/internal/job"
	"github.com/apgupta3091/job-runner/internal/worker"
)

func main() {
	// Setup store and worker pool
	store := job.NewStore(100)
	pool := worker.NewPool(5, store.Queue(), store)

	// Start workers
	pool.Start()

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("POST /jobs", createJobHandler(store))
	mux.HandleFunc("GET /jobs/{id}", getJobHandler(store))

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Run server in goroutine
	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received")

	//Graceful shutdown sequence
}
