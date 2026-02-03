package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rpi-metrics/internal/collectors"
	"rpi-metrics/internal/metrics"
)

//go:embed frontend/*
var frontendFS embed.FS

type MetricsResponse struct {
	Timestamp time.Time                   `json:"timestamp"`
	Metrics   map[string][]metrics.Sample `json:"metrics"`
	Error     string                      `json:"error,omitempty"`
}

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()

	// Initialize collectors
	cpuTemp := &collectors.CPUTempSysfs{}
	cpuUtil := &collectors.CPUUtilizationProcfs{}
	cpuCooling := &collectors.CPUCoolingDevicefs{}
	storage := &collectors.StorageStatfs{Paths: []string{"/", "/boot"}}

	allCollectors := []metrics.Collector{cpuTemp, cpuUtil, cpuCooling, storage}

	// Create router
	mux := http.NewServeMux()

	// API endpoint for metrics
	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ctx := r.Context()
		response := MetricsResponse{
			Timestamp: time.Now().UTC(),
			Metrics:   make(map[string][]metrics.Sample),
		}

		for _, c := range allCollectors {
			samples, err := c.Collect(ctx)
			if err != nil {
				log.Printf("collector %s error: %v", c.ID(), err)
				continue
			}
			response.Metrics[c.ID()] = samples
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("json encode error: %v", err)
		}
	})

	// Serve frontend static files
	frontendContent, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("failed to get frontend subdirectory: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendContent)))

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Starting server on http://localhost:%d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
