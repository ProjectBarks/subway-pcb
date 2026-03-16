package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/api"
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	pollInterval := flag.Duration("poll-interval", 15*time.Second, "Feed poll interval")
	ledMapPath := flag.String("led-map", "led_map.json", "Path to led_map.json")
	visualizerPath := flag.String("visualizer", "visualizer.html", "Path to visualizer HTML (empty to disable)")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Printf("subway-server starting (port=%d, poll-interval=%s)", *port, *pollInterval)

	// Create aggregator with 10-second train persistence.
	aggregator := mta.NewAggregator(10 * time.Second)

	// Create cancellable context for feed pollers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start all feed pollers.
	mta.StartAllPollers(ctx, aggregator, *pollInterval)

	// Load LED map for pixel rendering.
	ledMap, err := api.LoadLEDMap(*ledMapPath)
	if err != nil {
		log.Fatalf("Failed to load LED map: %v", err)
	}
	pixelRenderer := api.NewPixelRenderer(ledMap)

	// Set up HTTP server.
	apiServer := api.NewServer(aggregator, pixelRenderer, *visualizerPath)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      apiServer.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in a goroutine.
	go func() {
		log.Printf("HTTP server listening on :%d", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %v, shutting down...", sig)

	// Cancel feed pollers.
	cancel()

	// Graceful HTTP shutdown with 5-second timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("subway-server stopped")
}
