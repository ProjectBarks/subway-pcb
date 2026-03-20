package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/ProjectBarks/subway-pcb/service/ui"
)

func main() {
	outDir := "dist"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	f, err := os.Create(filepath.Join(outDir, "index.html"))
	if err != nil {
		log.Fatalf("create index.html: %v", err)
	}
	defer f.Close()

	if err := ui.Landing().Render(context.Background(), f); err != nil {
		log.Fatalf("render: %v", err)
	}

	log.Printf("wrote %s/index.html", outDir)
}
