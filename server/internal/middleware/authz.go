package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ProjectBarks/subway-pcb/server/internal/store"
)

// RequireAuth rejects requests with no user in context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects requests from non-admin users.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil || !user.IsAdmin() {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireBoardAccess checks that the user has access to the board identified
// by the {mac} URL parameter, or is an admin.
func RequireBoardAccess(s store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			// Admins bypass access check
			if user.IsAdmin() {
				next.ServeHTTP(w, r)
				return
			}

			mac := chi.URLParam(r, "mac")
			if mac == "" {
				http.Error(w, `{"error":"missing device MAC"}`, http.StatusBadRequest)
				return
			}

			has, err := s.HasAccess(mac, user.Email)
			if err != nil {
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}
			if !has {
				http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
