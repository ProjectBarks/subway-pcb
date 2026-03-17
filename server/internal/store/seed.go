package store

import (
	"log"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

// SeedThemes ensures all provided built-in themes exist in the store.
func SeedThemes(s Store, themes []model.Theme) error {
	for _, theme := range themes {
		existing, err := s.GetTheme(theme.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			log.Printf("store: seeding built-in theme %q (mode=%s)", theme.Name, theme.ModeName)
			if err := s.CreateTheme(&theme); err != nil {
				return err
			}
		}
	}
	return nil
}
