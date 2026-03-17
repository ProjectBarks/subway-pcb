package middleware

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/store"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	EnforceAuth bool
	AdminEmail  string
}

// AuthConfigFromEnv reads auth configuration from environment variables.
func AuthConfigFromEnv() AuthConfig {
	return AuthConfig{
		EnforceAuth: os.Getenv("ENFORCE_AUTH") == "true",
		AdminEmail:  os.Getenv("ADMIN_EMAIL"),
	}
}

// Auth returns middleware that identifies the current user.
// In local mode (no auth headers and ENFORCE_AUTH not set), everyone is an implicit admin.
func Auth(s store.Store, cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email := r.Header.Get("X-Forwarded-Email")
			name := r.Header.Get("X-Forwarded-User")

			if email != "" {
				// Authenticated via oauth2-proxy
				role := "user"
				if cfg.AdminEmail != "" && email == cfg.AdminEmail {
					role = "admin"
				}

				user := &model.User{
					Email:    email,
					Name:     name,
					Role:     role,
					LastSeen: time.Now(),
				}

				// Upsert user in store
				existing, _ := s.GetUser(email)
				if existing == nil {
					user.CreatedAt = time.Now()
				} else {
					user.CreatedAt = existing.CreatedAt
					// Preserve admin role if already set
					if existing.Role == "admin" {
						user.Role = "admin"
					}
				}
				if err := s.UpsertUser(user); err != nil {
					log.Printf("auth: failed to upsert user %s: %v", email, err)
				}

				ctx := WithUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if cfg.EnforceAuth {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			// Local mode — implicit admin
			localUser := &model.User{
				Email: "local@localhost",
				Name:  "Local User",
				Role:  "admin",
			}
			ctx := WithUser(r.Context(), localUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
