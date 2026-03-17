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
	"github.com/ProjectBarks/subway-pcb/server/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/server/internal/mode"
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
	"github.com/ProjectBarks/subway-pcb/server/internal/store"
	"github.com/ProjectBarks/subway-pcb/server/internal/ui"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	pollInterval := flag.Duration("poll-interval", 15*time.Second, "Feed poll interval")
	ledMapPath := flag.String("led-map", "led_map.json", "Path to led_map.json")
	dataDir := flag.String("data-dir", "data", "Data directory for bbolt database")
	templateDir := flag.String("template-dir", "templates", "Path to templates directory")
	_ = flag.String("visualizer", "", "deprecated")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Printf("subway-server starting (port=%d, poll-interval=%s)", *port, *pollInterval)

	// Auth config
	authCfg := middleware.AuthConfigFromEnv()
	if authCfg.EnforceAuth && authCfg.AdminEmail == "" {
		log.Fatal("ADMIN_EMAIL must be set when ENFORCE_AUTH=true")
	}

	// Initialize store (bbolt or MySQL based on env)
	db, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer db.Close()

	// Seed built-in themes
	if err := store.SeedBuiltInThemes(db); err != nil {
		log.Fatalf("Failed to seed themes: %v", err)
	}

	// Initialize mode registry
	modeRegistry := mode.NewRegistry()
	modeRegistry.Register(&mode.TrackMode{})
	modeRegistry.Register(&mode.SnakeMode{})

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
	pixelRenderer.SetDeps(db, modeRegistry)

	// Initialize template renderer
	renderer, err := ui.NewRenderer(*templateDir)
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Set up HTTP server.
	apiServer := api.NewServer(api.ServerConfig{
		Aggregator:    aggregator,
		PixelRenderer: pixelRenderer,
		Store:         db,
		ModeRegistry:  modeRegistry,
		Renderer:      renderer,
		AuthConfig:    authCfg,
	})

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

	// Close store.
	db.Close()

	// Graceful HTTP shutdown with 5-second timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("subway-server stopped")
}
