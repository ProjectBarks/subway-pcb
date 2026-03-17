package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/ProjectBarks/subway-pcb/server/internal/middleware"
	"github.com/ProjectBarks/subway-pcb/server/internal/mode"
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
	"github.com/ProjectBarks/subway-pcb/server/internal/store"
	"github.com/ProjectBarks/subway-pcb/server/internal/ui"
	"google.golang.org/protobuf/encoding/protojson"
)

// ServerConfig holds all dependencies for the HTTP server.
type ServerConfig struct {
	Aggregator    *mta.Aggregator
	PixelRenderer *PixelRenderer
	Store         store.Store
	ModeRegistry  *mode.Registry
	Renderer      *ui.Renderer
	AuthConfig    middleware.AuthConfig
}

// Server is the HTTP API server.
type Server struct {
	aggregator    *mta.Aggregator
	pixelRenderer *PixelRenderer
	store         store.Store
	modes         *mode.Registry
	renderer      *ui.Renderer
	authConfig    middleware.AuthConfig
	startTime     time.Time
	router        chi.Router
}

// NewServer creates a new API server with all dependencies.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		aggregator:    cfg.Aggregator,
		pixelRenderer: cfg.PixelRenderer,
		store:         cfg.Store,
		modes:         cfg.ModeRegistry,
		renderer:      cfg.Renderer,
		authConfig:    cfg.AuthConfig,
		startTime:     time.Now(),
	}
	s.buildRouter()
	return s
}

func (s *Server) buildRouter() {
	r := chi.NewRouter()
	r.Use(chimw.Logger, chimw.Recoverer)

	// Open endpoints (firmware + health + static assets)
	r.Get("/api/v1/pixels", s.handlePixels)
	r.Get("/api/v1/state", s.handleState)
	r.Get("/health", s.handleHealth)

	// Static files
	staticDir := http.Dir("static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticDir)))

	// Authenticated routes
	authMW := middleware.Auth(s.store, s.authConfig)
	requireBoardAccess := middleware.RequireBoardAccess(s.store)

	r.Group(func(r chi.Router) {
		r.Use(authMW)

		r.Get("/", s.handleDashboard)
		r.Get("/partials/board-list", s.handleBoardListPartial)

		// Board view (per-user access check — admins bypass)
		r.Route("/boards/{mac}", func(r chi.Router) {
			r.Use(requireBoardAccess)
			r.Get("/", s.handleBoardView)
			r.Put("/mode", s.handleSetMode)
			r.Put("/theme", s.handleSetTheme)
			r.Put("/name", s.handleSetName)
			r.Get("/preview", s.handleBoardPreview)
			// Access management
			r.Post("/access", s.handleGrantAccess)
			r.Delete("/access/{email}", s.handleRevokeAccess)
		})

		// Theme API
		r.Get("/api/v1/themes", s.handleListThemes)
		r.Post("/api/v1/themes", s.handleCreateTheme)
		r.Put("/api/v1/themes/{id}", s.handleUpdateTheme)
		r.Delete("/api/v1/themes/{id}", s.handleDeleteTheme)

		// Admin-only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin)
			r.Get("/api/v1/users", s.handleListUsers)
		})
	})

	s.router = r
}

// Handler returns the http.Handler for this server.
func (s *Server) Handler() http.Handler {
	return s.router
}

// --- Dashboard ---

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var boards []model.Device
	var err error
	if user.IsAdmin() {
		boards, err = s.store.ListDevices()
	} else {
		boards, err = s.store.ListDevicesByUser(user.Email)
	}
	if err != nil {
		log.Printf("api: dashboard error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Load themes for color swatches
	type boardCard struct {
		Device model.Device
		Theme  *model.Theme
	}
	cards := make([]boardCard, len(boards))
	for i, d := range boards {
		cards[i].Device = d
		if d.ThemeID != "" {
			t, _ := s.store.GetTheme(d.ThemeID)
			cards[i].Theme = t
		}
	}

	data := map[string]any{
		"User":      user,
		"Boards":    boards,
		"Cards":     cards,
		"ActiveMAC": "",
	}

	if isHTMX(r) {
		s.renderer.Render(w, "dashboard_content", data)
		return
	}
	s.renderer.Render(w, "dashboard", data)
}

func (s *Server) handleBoardListPartial(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	var boards []model.Device
	if user.IsAdmin() {
		boards, _ = s.store.ListDevices()
	} else {
		boards, _ = s.store.ListDevicesByUser(user.Email)
	}

	type boardCard struct {
		Device model.Device
		Theme  *model.Theme
	}
	cards := make([]boardCard, len(boards))
	for i, d := range boards {
		cards[i].Device = d
		if d.ThemeID != "" {
			t, _ := s.store.GetTheme(d.ThemeID)
			cards[i].Theme = t
		}
	}

	data := map[string]any{
		"User":  user,
		"Cards": cards,
	}
	s.renderer.Render(w, "board_grid", data)
}

// --- Board View ---

func (s *Server) handleBoardView(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	user := middleware.UserFromContext(r.Context())

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	theme, _ := s.store.GetTheme(device.ThemeID)
	if theme == nil {
		theme, _ = s.store.GetTheme("classic-mta")
	}

	themes, _ := s.store.ListThemes()
	access, _ := s.store.ListAccessByDevice(mac)

	// Get all boards for nav
	var boards []model.Device
	if user.IsAdmin() {
		boards, _ = s.store.ListDevices()
	} else {
		boards, _ = s.store.ListDevicesByUser(user.Email)
	}

	// Sort route colors for display
	routeKeys := sortedRouteKeys(theme.RouteColors)

	data := map[string]any{
		"User":      user,
		"Device":    device,
		"Theme":     theme,
		"Themes":    themes,
		"Access":    access,
		"Modes":     s.modes.List(),
		"Boards":    boards,
		"ActiveMAC": mac,
		"RouteKeys": routeKeys,
	}

	s.renderer.Render(w, "board", data)
}

func (s *Server) handleSetMode(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()
	modeName := r.FormValue("mode")

	if _, ok := s.modes.Get(modeName); !ok {
		http.Error(w, "unknown mode", http.StatusBadRequest)
		return
	}

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	device.Mode = modeName
	if err := s.store.UpsertDevice(device); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	// Invalidate pixel cache for this device
	s.pixelRenderer.Invalidate()

	s.renderControls(w, r, mac)
}

func (s *Server) handleSetTheme(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()
	themeID := r.FormValue("theme_id")

	if _, err := s.store.GetTheme(themeID); err != nil {
		http.Error(w, "theme not found", http.StatusBadRequest)
		return
	}

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	device.ThemeID = themeID
	if err := s.store.UpsertDevice(device); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	s.pixelRenderer.Invalidate()
	s.renderControls(w, r, mac)
}

func (s *Server) handleSetName(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	r.ParseForm()
	name := r.FormValue("name")

	device, err := s.store.GetDevice(mac)
	if err != nil || device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	device.Name = name
	if err := s.store.UpsertDevice(device); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	s.renderControls(w, r, mac)
}

func (s *Server) handleBoardPreview(w http.ResponseWriter, r *http.Request) {
	// Returns partial HTML for the board preview area
	mac := chi.URLParam(r, "mac")
	device, _ := s.store.GetDevice(mac)
	if device == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="text-sm text-gray-400">Preview for %s</div>`, device.Name)
}

// --- Access Management ---

func (s *Server) handleGrantAccess(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	user := middleware.UserFromContext(r.Context())
	r.ParseForm()
	email := r.FormValue("email")

	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	access := &model.DeviceAccess{
		MAC:       mac,
		UserEmail: email,
		GrantedBy: user.Email,
		GrantedAt: time.Now(),
	}

	if err := s.store.GrantAccess(access); err != nil {
		http.Error(w, "grant failed", http.StatusInternalServerError)
		return
	}

	s.renderAccessPanel(w, mac)
}

func (s *Server) handleRevokeAccess(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")
	email := chi.URLParam(r, "email")

	if err := s.store.RevokeAccess(mac, email); err != nil {
		http.Error(w, "revoke failed", http.StatusInternalServerError)
		return
	}

	s.renderAccessPanel(w, mac)
}

func (s *Server) renderAccessPanel(w http.ResponseWriter, mac string) {
	access, _ := s.store.ListAccessByDevice(mac)
	device, _ := s.store.GetDevice(mac)
	data := map[string]any{
		"Device": device,
		"Access": access,
	}
	s.renderer.Render(w, "device_access", data)
}

func (s *Server) renderControls(w http.ResponseWriter, r *http.Request, mac string) {
	user := middleware.UserFromContext(r.Context())
	device, _ := s.store.GetDevice(mac)
	theme, _ := s.store.GetTheme(device.ThemeID)
	if theme == nil {
		theme, _ = s.store.GetTheme("classic-mta")
	}
	themes, _ := s.store.ListThemes()
	access, _ := s.store.ListAccessByDevice(mac)
	routeKeys := sortedRouteKeys(theme.RouteColors)

	data := map[string]any{
		"User":      user,
		"Device":    device,
		"Theme":     theme,
		"Themes":    themes,
		"Access":    access,
		"Modes":     s.modes.List(),
		"RouteKeys": routeKeys,
	}
	s.renderer.Render(w, "board_controls", data)
}

// --- Theme API ---

func (s *Server) handleListThemes(w http.ResponseWriter, r *http.Request) {
	themes, err := s.store.ListThemes()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, themes)
}

func (s *Server) handleCreateTheme(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var theme model.Theme
	if isHTMX(r) {
		r.ParseForm()
		theme.Name = r.FormValue("name")
		// Parse route colors from form
		theme.RouteColors = parseRouteColorsFromForm(r)
	} else {
		if err := json.NewDecoder(r.Body).Decode(&theme); err != nil {
			jsonError(w, "invalid body", http.StatusBadRequest)
			return
		}
	}

	theme.ID = fmt.Sprintf("custom-%d", time.Now().UnixMilli())
	theme.OwnerEmail = user.Email
	theme.CreatedAt = time.Now()
	theme.UpdatedAt = time.Now()

	if err := s.store.CreateTheme(&theme); err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "themeCreated")
		fmt.Fprintf(w, `<div class="text-green-400 text-sm">Theme "%s" saved</div>`, theme.Name)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, theme)
}

func (s *Server) handleUpdateTheme(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, err := s.store.GetTheme(id)
	if err != nil || existing == nil {
		jsonError(w, "theme not found", http.StatusNotFound)
		return
	}

	if existing.IsBuiltIn {
		jsonError(w, "cannot modify built-in theme", http.StatusForbidden)
		return
	}

	if existing.OwnerEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
		return
	}

	if isHTMX(r) {
		r.ParseForm()
		if name := r.FormValue("name"); name != "" {
			existing.Name = name
		}
		existing.RouteColors = parseRouteColorsFromForm(r)
	} else {
		var update model.Theme
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			jsonError(w, "invalid body", http.StatusBadRequest)
			return
		}
		if update.Name != "" {
			existing.Name = update.Name
		}
		if update.RouteColors != nil {
			existing.RouteColors = update.RouteColors
		}
	}

	existing.UpdatedAt = time.Now()
	if err := s.store.UpdateTheme(existing); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	s.pixelRenderer.Invalidate()

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "themeUpdated")
		fmt.Fprintf(w, `<div class="text-green-400 text-sm">Theme updated</div>`)
		return
	}
	jsonResponse(w, existing)
}

func (s *Server) handleDeleteTheme(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.UserFromContext(r.Context())

	existing, err := s.store.GetTheme(id)
	if err != nil || existing == nil {
		jsonError(w, "theme not found", http.StatusNotFound)
		return
	}

	if existing.IsBuiltIn {
		jsonError(w, "cannot delete built-in theme", http.StatusForbidden)
		return
	}

	if existing.OwnerEmail != user.Email && !user.IsAdmin() {
		jsonError(w, "access denied", http.StatusForbidden)
		return
	}

	if err := s.store.DeleteTheme(id); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "themeDeleted")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Users (Admin) ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, users)
}

// --- State & Health (unchanged core logic) ---

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")

	if format == "json" {
		state := s.aggregator.GetState()
		if state == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}

		opts := protojson.MarshalOptions{
			EmitUnpopulated: true,
			UseProtoNames:   true,
		}
		data, err := opts.Marshal(state)
		if err != nil {
			log.Printf("api: JSON marshal error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	data := s.aggregator.GetStateBytes()
	if data == nil {
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

type healthResponse struct {
	Status        string  `json:"status"`
	Uptime        string  `json:"uptime"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	LastUpdate    string  `json:"last_update"`
	StationCount  int     `json:"station_count"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(s.startTime)
	lastUpdate := s.aggregator.LastUpdate()

	lastUpdateStr := "never"
	if !lastUpdate.IsZero() {
		lastUpdateStr = fmt.Sprintf("%ds ago", int(time.Since(lastUpdate).Seconds()))
	}

	resp := healthResponse{
		Status:        "ok",
		Uptime:        uptime.Round(time.Second).String(),
		UptimeSeconds: uptime.Seconds(),
		LastUpdate:    lastUpdateStr,
		StationCount:  s.aggregator.StationCount(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// --- Helpers ---

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"error": msg})
}

func parseRouteColorsFromForm(r *http.Request) map[string][3]uint8 {
	colors := make(map[string][3]uint8)
	for key, vals := range r.Form {
		if len(key) > 6 && key[:6] == "color_" && len(vals) > 0 {
			routeKey := key[6:]
			hex := vals[0]
			if len(hex) == 7 && hex[0] == '#' {
				var r, g, b uint8
				fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
				colors[routeKey] = [3]uint8{r, g, b}
			}
		}
	}
	return colors
}

func sortedRouteKeys(m map[string][3]uint8) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

