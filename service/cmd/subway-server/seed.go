package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
	"github.com/ProjectBarks/subway-pcb/service/internal/store"
)

// seedBuiltinPlugins upserts all built-in plugins (and their default presets)
// into the store so that every code path can look up plugins from the DB alone.
func seedBuiltinPlugins(st store.Store, registry *plugin.Registry) error {
	now := time.Now()
	for _, p := range registry.List() {
		configFieldsJSON, err := json.Marshal(p.ConfigFields())
		if err != nil {
			return err
		}

		dbPlugin := &model.Plugin{
			ID:               p.Name(),
			Name:             p.Name(),
			Type:             "builtin",
			Description:      p.Description(),
			LuaSource:        p.LuaSource(),
			ConfigFields:     configFieldsJSON,
			RequiredFeatures: p.RequiredFeatures(),
			IsPublished:      true,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		existing, _ := st.GetPlugin(p.Name())
		if existing == nil {
			log.Printf("seed: creating built-in plugin %q", p.Name())
			if err := st.CreatePlugin(dbPlugin); err != nil {
				return err
			}
		} else {
			// Update existing built-in plugin to pick up code changes
			dbPlugin.CreatedAt = existing.CreatedAt
			dbPlugin.Installs = existing.Installs
			if err := st.UpdatePlugin(dbPlugin); err != nil {
				return err
			}
		}

		// Seed default presets
		for _, preset := range p.DefaultPresets() {
			preset.IsBuiltIn = true
			existingPreset, _ := st.GetPreset(preset.ID)
			if existingPreset == nil {
				log.Printf("seed: creating built-in preset %q (plugin=%s)", preset.Name, preset.PluginName)
				if err := st.CreatePreset(&preset); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
