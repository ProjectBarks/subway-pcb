package store

import (
	"log"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

// BuiltInThemes returns the predefined themes to seed on startup.
func BuiltInThemes() []model.Theme {
	now := time.Now()
	return []model.Theme{
		{
			ID: "classic-mta", Name: "Classic MTA", IsBuiltIn: true,
			CreatedAt: now, UpdatedAt: now,
			RouteColors: map[string][3]uint8{
				"ROUTE_1": {255, 0, 0}, "ROUTE_2": {255, 0, 0}, "ROUTE_3": {255, 0, 0},
				"ROUTE_4": {0, 147, 60}, "ROUTE_5": {0, 147, 60}, "ROUTE_6": {0, 147, 60},
				"ROUTE_7": {185, 51, 173},
				"ROUTE_A": {0, 57, 166}, "ROUTE_C": {0, 57, 166}, "ROUTE_E": {0, 57, 166},
				"ROUTE_B": {255, 99, 25}, "ROUTE_D": {255, 99, 25}, "ROUTE_F": {255, 99, 25}, "ROUTE_M": {255, 99, 25},
				"ROUTE_G":  {108, 190, 69},
				"ROUTE_J":  {153, 102, 51}, "ROUTE_Z": {153, 102, 51},
				"ROUTE_L":  {167, 169, 172},
				"ROUTE_N":  {252, 204, 10}, "ROUTE_Q": {252, 204, 10}, "ROUTE_R": {252, 204, 10}, "ROUTE_W": {252, 204, 10},
				"ROUTE_S":  {128, 129, 131}, "ROUTE_FS": {128, 129, 131}, "ROUTE_GS": {128, 129, 131},
				"ROUTE_SI": {0, 57, 166},
			},
		},
		{
			ID: "neon", Name: "Neon", IsBuiltIn: true,
			CreatedAt: now, UpdatedAt: now,
			RouteColors: map[string][3]uint8{
				"ROUTE_1": {255, 0, 80}, "ROUTE_2": {255, 0, 80}, "ROUTE_3": {255, 0, 80},
				"ROUTE_4": {0, 255, 100}, "ROUTE_5": {0, 255, 100}, "ROUTE_6": {0, 255, 100},
				"ROUTE_7": {200, 0, 255},
				"ROUTE_A": {0, 100, 255}, "ROUTE_C": {0, 100, 255}, "ROUTE_E": {0, 100, 255},
				"ROUTE_B": {255, 120, 0}, "ROUTE_D": {255, 120, 0}, "ROUTE_F": {255, 120, 0}, "ROUTE_M": {255, 120, 0},
				"ROUTE_G":  {0, 255, 200},
				"ROUTE_J":  {255, 200, 0}, "ROUTE_Z": {255, 200, 0},
				"ROUTE_L":  {200, 200, 255},
				"ROUTE_N":  {255, 255, 0}, "ROUTE_Q": {255, 255, 0}, "ROUTE_R": {255, 255, 0}, "ROUTE_W": {255, 255, 0},
				"ROUTE_S":  {180, 180, 255}, "ROUTE_FS": {180, 180, 255}, "ROUTE_GS": {180, 180, 255},
				"ROUTE_SI": {0, 150, 255},
			},
		},
		{
			ID: "pastel", Name: "Pastel", IsBuiltIn: true,
			CreatedAt: now, UpdatedAt: now,
			RouteColors: map[string][3]uint8{
				"ROUTE_1": {255, 150, 150}, "ROUTE_2": {255, 150, 150}, "ROUTE_3": {255, 150, 150},
				"ROUTE_4": {150, 220, 170}, "ROUTE_5": {150, 220, 170}, "ROUTE_6": {150, 220, 170},
				"ROUTE_7": {210, 170, 220},
				"ROUTE_A": {150, 170, 220}, "ROUTE_C": {150, 170, 220}, "ROUTE_E": {150, 170, 220},
				"ROUTE_B": {255, 190, 150}, "ROUTE_D": {255, 190, 150}, "ROUTE_F": {255, 190, 150}, "ROUTE_M": {255, 190, 150},
				"ROUTE_G":  {170, 230, 180},
				"ROUTE_J":  {210, 190, 160}, "ROUTE_Z": {210, 190, 160},
				"ROUTE_L":  {200, 200, 210},
				"ROUTE_N":  {255, 240, 170}, "ROUTE_Q": {255, 240, 170}, "ROUTE_R": {255, 240, 170}, "ROUTE_W": {255, 240, 170},
				"ROUTE_S":  {190, 190, 200}, "ROUTE_FS": {190, 190, 200}, "ROUTE_GS": {190, 190, 200},
				"ROUTE_SI": {150, 180, 220},
			},
		},
		{
			ID: "monochrome", Name: "Monochrome", IsBuiltIn: true,
			CreatedAt: now, UpdatedAt: now,
			RouteColors: map[string][3]uint8{
				"ROUTE_1": {255, 255, 255}, "ROUTE_2": {255, 255, 255}, "ROUTE_3": {255, 255, 255},
				"ROUTE_4": {255, 255, 255}, "ROUTE_5": {255, 255, 255}, "ROUTE_6": {255, 255, 255},
				"ROUTE_7": {255, 255, 255},
				"ROUTE_A": {255, 255, 255}, "ROUTE_C": {255, 255, 255}, "ROUTE_E": {255, 255, 255},
				"ROUTE_B": {255, 255, 255}, "ROUTE_D": {255, 255, 255}, "ROUTE_F": {255, 255, 255}, "ROUTE_M": {255, 255, 255},
				"ROUTE_G":  {255, 255, 255},
				"ROUTE_J":  {255, 255, 255}, "ROUTE_Z": {255, 255, 255},
				"ROUTE_L":  {255, 255, 255},
				"ROUTE_N":  {255, 255, 255}, "ROUTE_Q": {255, 255, 255}, "ROUTE_R": {255, 255, 255}, "ROUTE_W": {255, 255, 255},
				"ROUTE_S":  {255, 255, 255}, "ROUTE_FS": {255, 255, 255}, "ROUTE_GS": {255, 255, 255},
				"ROUTE_SI": {255, 255, 255},
			},
		},
	}
}

// SeedBuiltInThemes ensures all built-in themes exist in the store.
func SeedBuiltInThemes(s Store) error {
	for _, theme := range BuiltInThemes() {
		existing, err := s.GetTheme(theme.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			log.Printf("store: seeding built-in theme %q", theme.Name)
			if err := s.CreateTheme(&theme); err != nil {
				return err
			}
		}
	}
	return nil
}
