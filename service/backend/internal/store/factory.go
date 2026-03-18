package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ProjectBarks/subway-pcb/server/internal/store/bolt"
	"github.com/ProjectBarks/subway-pcb/server/internal/store/mysql"
)

// New creates a Store based on environment configuration.
// If MYSQL_DSN is set, uses MySQL. Otherwise uses bbolt at dataDir/subway.db.
func New(dataDir string) (Store, error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn != "" {
		log.Printf("store: using MySQL backend")
		return mysql.New(dsn)
	}

	dbPath := filepath.Join(dataDir, "subway.db")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	log.Printf("store: using bbolt backend at %s", dbPath)
	return bolt.New(dbPath)
}
