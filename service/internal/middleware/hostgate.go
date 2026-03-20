package middleware

import (
	"log"
	"net/http"
	"regexp"
)

// HostRestriction only allows requests whose Host header matches the given
// regex pattern. Non-matching hosts receive a 404. Routes outside this
// middleware remain accessible on all hosts. If pattern is empty the
// middleware is a no-op. Panics if the pattern is invalid.
func HostRestriction(pattern string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if pattern == "" {
			return next
		}
		re := regexp.MustCompile(pattern)
		log.Printf("hostgate: allowing hosts matching %s", pattern)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !re.MatchString(r.Host) {
				log.Printf("hostgate: blocked %s %s from host %s", r.Method, r.URL.Path, r.Host)
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
