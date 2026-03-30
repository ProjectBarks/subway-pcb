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

	"path/filepath"

	"github.com/ProjectBarks/subway-pcb/service/internal/api"
	"github.com/ProjectBarks/subway-pcb/service/internal/manifest"
	"github.com/ProjectBarks/subway-pcb/service/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/service/internal/mta"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin/snake"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin/track"
	"github.com/ProjectBarks/subway-pcb/service/internal/store"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	pollInterval := flag.Duration("poll-interval", 15*time.Second, "Feed poll interval")
	boardsDir := flag.String("boards-dir", "public/boards", "Directory containing versioned board definitions")
	dataDir := flag.String("data-dir", "data", "Data directory for bbolt database")
	staticDir := flag.String("static-dir", "", "Path to static assets directory (serves /static/)")
	devMode := flag.Bool("dev", false, "Enable dev-only routes (e.g. /landing)")
	_ = flag.String("led-map", "", "deprecated, use --boards-dir")
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

	// Initialize plugin registry
	pluginRegistry := plugin.NewRegistry()
	pluginRegistry.Register(&track.Plugin{})
	pluginRegistry.Register(&snake.Plugin{})

	// Seed built-in plugins and their presets into the store
	if err := seedBuiltinPlugins(db, pluginRegistry); err != nil {
		log.Fatalf("Failed to seed built-in plugins: %v", err)
	}

	// Load Vite asset manifest for content-hashed filenames
	if *staticDir != "" {
		if err := manifest.Load(filepath.Join(*staticDir, "dist")); err != nil {
			log.Printf("manifest: %v (using unhashed paths)", err)
		}
	}

	// Load all versioned board definitions
	boards, err := api.LoadAllBoards(*boardsDir)
	if err != nil {
		log.Fatalf("Failed to load boards: %v", err)
	}

	// Validate board defaults
	for key, b := range boards {
		if b.Manifest.DefaultPlugin == "" {
			log.Fatalf("board %s has empty DefaultPlugin", key)
		}
		if b.Manifest.DefaultPreset == "" {
			log.Fatalf("board %s has empty DefaultPreset", key)
		}
	}

	// Create aggregator.
	aggregator := mta.NewAggregator()

	// Create cancellable context for feed pollers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start all feed pollers.
	mta.StartAllPollers(ctx, aggregator, *pollInterval)

	// Set up HTTP server.
	apiServer := api.NewServer(api.ServerConfig{
		Aggregator: aggregator,
		Store:      db,
		Boards:     boards,
		AuthConfig: authCfg,
		StaticDir:  *staticDir,
		DevMode:    *devMode,
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
