package store

import (
	"log"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
)

// SeedPresets ensures all provided built-in presets exist in the store.
func SeedPresets(s Store, presets []model.Preset) error {
	for _, preset := range presets {
		existing, err := s.GetPreset(preset.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			log.Printf("store: seeding built-in preset %q (plugin=%s)", preset.Name, preset.PluginName)
			if err := s.CreatePreset(&preset); err != nil {
				return err
			}
		}
	}
	return nil
}
