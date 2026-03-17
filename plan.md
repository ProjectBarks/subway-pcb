# Subway PCB Web UI — Implementation Plan

## Context

The server currently has no database, no auth, and a minimal static HTML/JS viewer. The goal is to build a gorgeous, composable Go-powered dashboard with Tailwind + HTMX that supports:

- **Dual database**: bbolt locally, MySQL when `MYSQL_DSN` is provided
- **Auth**: oauth2-proxy headers when present; local mode = all admin
- **Device tracking**: Store board MAC addresses, assign boards to users
- **Board modes**: Pluggable rendering (track, snake, extensible)
- **Themes**: Predefined + custom color schemes, saveable, with live draft preview
- **Unified board view**: Mode selection, theme editing, and live preview all in one gorgeous page
- **Access control**: Users see only boards they've been given access to; admins see all + extra controls

---

## Phase 1: Data Models & Store Interface

### New files

**`server/internal/model/model.go`** — Plain domain structs:
```go
type Device struct {
    MAC, Name, Mode, ThemeID, FirmwareVer string
    LastSeen, CreatedAt time.Time
}
type DeviceAccess struct {
    MAC, UserEmail string
    GrantedBy      string    // admin email who granted access
    GrantedAt      time.Time
}
type Theme struct {
    ID, Name, OwnerEmail string
    IsBuiltIn bool
    RouteColors map[string][3]uint8  // "ROUTE_1" -> [255,0,0]
    CreatedAt, UpdatedAt time.Time
}
type User struct {
    Email, Name, Role string  // Role: "admin" | "user"
    CreatedAt, LastSeen time.Time
}
```

**`server/internal/store/store.go`** — Repository interface:
- **Devices**: `GetDevice`, `ListDevices`, `ListDevicesByUser`, `UpsertDevice`, `UpdateDeviceLastSeen`
- **Access**: `GrantAccess`, `RevokeAccess`, `ListAccessByDevice`, `ListAccessByUser`, `HasAccess`
- **Themes**: `GetTheme`, `ListThemes`, `ListThemesByOwner`, `CreateTheme`, `UpdateTheme`, `DeleteTheme`
- **Users**: `GetUser`, `UpsertUser`, `ListUsers`
- **Lifecycle**: `Close() error`

**`server/internal/store/factory.go`** — Selects backend:
- If `MYSQL_DSN` is set → MySQL backend
- Otherwise → bbolt at `{data-dir}/subway.db`

**`server/internal/store/seed.go`** — Seeds built-in themes on startup:
- **"Classic MTA"**: current `routeColors` from `pixels.go`
- **"Neon"**: high-saturation electric colors
- **"Pastel"**: soft muted tones
- **"Monochrome"**: white-only (all routes same color)

### bbolt backend — `server/internal/store/bolt/bolt.go`
- Buckets: `devices`, `device_access`, `themes`, `users`
- Values: JSON-encoded model structs
- Pure Go, CGO_ENABLED=0 compatible, zero config

### MySQL backend — `server/internal/store/mysql/mysql.go`

Uses **GORM** (`gorm.io/gorm` + `gorm.io/driver/mysql`) — provides auto-migration, associations, hooks, soft deletes, and query building out of the box. Models use GORM struct tags:

```go
// GORM auto-migrates these structs into MySQL tables
type DeviceModel struct {
    MAC         string `gorm:"primaryKey;size:17"`
    Name        string `gorm:"size:255"`
    Mode        string `gorm:"size:50;default:track"`
    ThemeID     string `gorm:"size:50"`
    FirmwareVer string `gorm:"size:50"`
    LastSeen    time.Time
    CreatedAt   time.Time
    Access      []DeviceAccessModel `gorm:"foreignKey:MAC;references:MAC"`
}
type DeviceAccessModel struct {
    MAC       string `gorm:"primaryKey;size:17"`
    UserEmail string `gorm:"primaryKey;size:255"`
    GrantedBy string `gorm:"size:255"`
    GrantedAt time.Time
}
type ThemeModel struct {
    ID          string `gorm:"primaryKey;size:50"`
    Name        string `gorm:"size:255"`
    OwnerEmail  string `gorm:"size:255"`
    IsBuiltIn   bool
    RouteColors datatypes.JSON  // gorm.io/datatypes for native JSON column
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
type UserModel struct {
    Email     string `gorm:"primaryKey;size:255"`
    Name      string `gorm:"size:255"`
    Role      string `gorm:"size:20;default:user"`
    CreatedAt time.Time
    LastSeen  time.Time
}
```

- `db.AutoMigrate()` on connect — no manual SQL needed
- GORM handles all CRUD, associations, and pagination
- bbolt backend still uses the raw `Store` interface with JSON encoding

---

## Phase 2: Mode System

### New files

**`server/internal/mode/mode.go`** — Interface + registry:
```go
type RenderContext struct {
    Aggregator *mta.Aggregator
    LEDMap     *LEDMap
    Theme      *model.Theme
    Device     *model.Device
    TotalLEDs  int
}
type Mode interface {
    Name() string
    Description() string
    Render(ctx RenderContext) ([]byte, error) // returns RGB byte array (len = TotalLEDs*3)
}
type Registry struct { modes map[string]Mode }
func (r *Registry) Register(m Mode)
func (r *Registry) Get(name string) (Mode, bool)
func (r *Registry) List() []Mode  // for UI to enumerate available modes
```

**`server/internal/mode/track.go`** — Extracts current `GetFrame` logic from `pixels.go`. Uses `ctx.Theme.RouteColors` instead of hardcoded `routeColors`.

**`server/internal/mode/snake.go`** — Stateless time-stepped snake. Position computed from `time.Now()` modulo path length. Uses LED adjacency from `leds.json`. Snake color from theme.

### Modified: `server/internal/api/pixels.go`
- `handlePixels` becomes per-device:
  1. Read `X-Device-ID` header (firmware already sends this)
  2. Look up or auto-register device in store (default: mode="track", theme="classic-mta")
  3. Get device's mode from registry + theme from store
  4. Call `mode.Render(ctx)` for per-device pixel generation
  5. Per-device cache keyed on `mac + mode + themeID + aggregator.lastUpdate`
- New query param: `GET /api/v1/pixels?device={mac}` for web UI live preview
- New endpoint: `GET /api/v1/pixels/preview?theme={json}` for live draft preview in theme editor

---

## Phase 3: Authentication & Authorization Middleware

### New files

**`server/internal/middleware/auth.go`** — Generic HTTP middleware:

```go
func Auth(store store.Store, cfg AuthConfig) func(http.Handler) http.Handler
```

Behavior:
- Check `X-Forwarded-Email` / `X-Forwarded-User` headers (set by oauth2-proxy)
- If headers present: upsert user in store, attach `*model.User` to request context
- If absent + `ENFORCE_AUTH=true`: return 401 JSON error
- If absent + `ENFORCE_AUTH` not set: create implicit admin user in context (local mode — everyone is admin)
- If `user.Email == ADMIN_EMAIL` env var: set role to `"admin"`

**`server/internal/middleware/authz.go`** — Authorization middleware:

```go
func RequireAdmin(next http.Handler) http.Handler  // 403 if not admin
func RequireAuth(next http.Handler) http.Handler   // 401 if no user in context
func RequireBoardAccess(store store.Store) func(http.Handler) http.Handler
    // checks user has access to the {mac} in URL param, or is admin
```

**`server/internal/middleware/context.go`** — `UserFromContext(ctx) *model.User`, `WithUser(ctx, user) context.Context`

### Env vars
| Variable | Purpose | Default |
|----------|---------|---------|
| `ENFORCE_AUTH` | `"true"` to require auth | "" (local mode) |
| `ADMIN_EMAIL` | Email that gets admin role (required if ENFORCE_AUTH=true) | "" |
| `MYSQL_DSN` | MySQL connection string | "" (use bbolt) |

### Middleware application to routes
- **No auth**: `/api/v1/pixels`, `/api/v1/state`, `/health`, `/static/*` (firmware + assets)
- **Auth**: `/`, `/boards/*`, `/api/v1/themes`
- **Auth + board access**: `/boards/{mac}/*` (view, mode, theme, name, access grant/revoke)
- **Auth + admin only**: `/api/v1/users`

Note: Access grant/revoke (`POST /boards/{mac}/access`, `DELETE /boards/{mac}/access/{email}`) uses `RequireBoardAccess` — anyone with existing access to a board can share it with others. Admins bypass the access check and can manage access for any device.

---

## Phase 4: Router & API Endpoints

### Modified: `server/internal/api/server.go`
Migrate from `http.ServeMux` to **`chi.Router`** (`github.com/go-chi/chi/v5`). Server struct gains `Store`, `ModeRegistry`, `Renderer`, `AuthConfig`.

```go
r := chi.NewRouter()
r.Use(middleware.Logger, middleware.Recoverer)

// Open endpoints (firmware + health)
r.Get("/api/v1/pixels", s.handlePixels)
r.Get("/api/v1/state", s.handleState)
r.Get("/health", s.handleHealth)
r.Handle("/static/*", staticFS)

// Authenticated routes
r.Group(func(r chi.Router) {
    r.Use(authMiddleware)

    r.Get("/", s.handleDashboard)

    // Board view (per-user access check — admins bypass)
    r.Route("/boards/{mac}", func(r chi.Router) {
        r.Use(requireBoardAccess) // checks user has access OR is admin
        r.Get("/", s.handleBoardView)
        r.Put("/mode", s.handleSetMode)        // HTMX
        r.Put("/theme", s.handleSetTheme)       // HTMX
        r.Put("/name", s.handleSetName)         // HTMX
        r.Get("/preview", s.handleBoardPreview) // HTMX partial
        // Access management — anyone with board access can share
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
        r.Use(requireAdmin)
        r.Get("/api/v1/users", s.handleListUsers)
    })
})
```

### API response pattern
- JSON endpoints: `{"data": ..., "error": "..."}` envelope
- HTMX endpoints (detected via `HX-Request` header): return HTML partials
- Board control endpoints (`/boards/{mac}/mode`, etc.) are HTMX-first — return updated partial HTML

---

## Phase 5: UI — Gorgeous Tailwind + HTMX Dashboard

### Design philosophy
- **Dark-first**: Deep blacks (#0a0a0a, #111) with subtle borders (#1e1e1e)
- **Gold accent**: `#c9a830` carried from existing header — used for active states, highlights
- **JetBrains Mono** font (already loaded in current UI)
- **Glassmorphism touches**: subtle `backdrop-blur`, translucent panels
- **Smooth transitions**: all interactive elements have CSS transitions
- **Responsive**: works on desktop and tablet

### Tech stack
- **Go `html/template`** with FuncMap: `colorHex`, `routeName`, `timeAgo`, `isAdmin`
- **Tailwind CSS** via CDN play script with custom config
- **HTMX** for all dynamic interactions (no full page reloads for controls)
- **Existing `board.js`** reused for live canvas previews
- **New `preview.js`** for client-side theme draft rendering

### Template structure
```
server/templates/
  layouts/
    base.html              — HTML skeleton, CDN imports, sidebar nav, content slot
  partials/
    nav.html               — Sidebar: logo, board list, theme link, admin section (if admin)
    board_card.html        — Dashboard card: name, mode pill, theme dots, online indicator
    board_controls.html    — Mode selector + theme selector + name edit (single partial)
    theme_form.html        — Route color grid with color pickers
    device_access.html     — User access list + add user form (admin only)
    toast.html             — Toast notification
  pages/
    dashboard.html         — Board card grid + empty state
    board.html             — Unified board view: live preview + controls + theme edit
    login_required.html    — Auth gate page
```

### Page: Dashboard (`/`)

Responsive grid of board cards. Everyone sees the same layout — admins just see all boards, users see boards they have access to.

Each **board card** shows:
- Board name (or truncated MAC if unnamed)
- Mode pill badge (e.g., "track" in a rounded pill)
- Theme color swatch (row of small colored dots for each route)
- Online/offline indicator: green glow dot if last seen < 30s ago
- Last seen timestamp ("2m ago")
- Click → navigates to `/boards/{mac}`

**Admin extras on dashboard**: count of total boards, unassigned boards highlighted with subtle border

Auto-refresh: `hx-get="/partials/board-list" hx-trigger="every 5s" hx-swap="innerHTML"`

**Empty state**: Centered message "No boards connected yet" with subtle animated dots

### Page: Board View (`/boards/{mac}`) — THE main page

This is the unified view where all board configuration happens. Two-column layout on desktop, stacked on mobile.

**Left column (60%)**: Live board preview
- Full `board.js` canvas showing the PCB with LEDs
- Fetches `/api/v1/pixels?device={mac}` every 1s via JS
- Decodes protobuf, renders on canvas
- Hover tooltips (station ID, RGB, strip/pixel)
- Status bar below canvas: online dot + train count + sequence number

**Right column (40%)**: Control panel — beautiful stacked cards with subtle glass effect

**Card 1 — Mode** (collapsible):
- Visual radio buttons for each registered mode
- Each option shows: mode name, description, icon
- Selecting a mode fires HTMX PUT, updates immediately
- Active mode has gold `#c9a830` ring indicator

**Card 2 — Theme** (collapsible, expanded by default):
- **Theme selector dropdown** at top — pick from built-in + saved custom themes
- Selecting a theme fires HTMX PUT, board preview updates within 1s
- **"Customize" toggle** — expands inline theme editor below:
  - Route color grid: route name | color picker | hex value | preview dot
  - **Live draft preview**: As any color picker changes, JS immediately re-renders the board canvas with draft colors (no server call). Implementation:
    1. Fetch `/api/v1/state?format=json` → current train positions
    2. Load `led_map.json` → LED-to-station mapping
    3. Client-side: map trains → stations → LEDs → apply draft colors
    4. Feed RGB array to `board.setPixels()` — instant visual feedback
    5. State auto-refreshes every 5s for current train data
  - "Save as new theme" button (name input + HTMX POST)
  - "Update theme" button (if editing existing custom theme)
  - Built-in themes cannot be overwritten, only cloned

**Card 3 — Device Info**:
- Board name (inline editable, HTMX PUT on blur)
- MAC address (copyable)
- Firmware version
- First seen / Last seen timestamps
- Online status with uptime

**Card 4 — Access** (visible to all users with board access):
- List of users with access to this board
- Each user row: email, granted by, "Remove" button (HTMX DELETE)
- "Add user" input + button at bottom (HTMX POST)
- Anyone with access can share the board with others or remove access
- Admins see this card on every board (even ones they don't explicitly have access to)

### Sidebar navigation
- Logo/title: "NYC SUBWAY PCB" in gold
- Board list: all accessible boards as nav items, each with online dot
- "Themes" link (goes to dashboard with theme management section)
- Admin section (if admin): shows total device count, link to full device list
- User info at bottom: email + role badge

### New static assets
**`server/static/preview.js`** — Client-side theme draft preview:
- Fetches `/api/v1/state?format=json` and `/static/leds.json` and `led_map.json`
- Exports `PreviewRenderer` class that takes a `Board` instance
- `previewRenderer.setThemeColors(routeColors)` → re-renders canvas with given colors
- Called on every color picker `input` event for instant feedback
- Falls back to server-rendered preview if state fetch fails

**`server/static/app.js`** — Updated to support `?device=` param for per-device preview

Existing `board.js`, `board.svg`, `leds.json` remain unchanged.

---

## Phase 6: Wiring — main.go Changes

### Modified: `server/cmd/subway-server/main.go`

Startup sequence:
1. Parse flags (existing: `-port`, `-poll-interval`, `-led-map`) + env vars
2. Initialize store via factory (bbolt or MySQL)
3. Seed built-in themes if not present
4. Initialize mode registry → register TrackMode, SnakeMode
5. Validate: if `ENFORCE_AUTH=true`, require `ADMIN_EMAIL` set (log.Fatal if missing)
6. Initialize template renderer from `templates/` dir
7. Create `api.Server` with `ServerConfig{Aggregator, LEDMap, Store, ModeRegistry, Renderer, AuthConfig}`
8. Start feed pollers (unchanged)
9. Start HTTP server (unchanged)

### Dockerfile updates
```dockerfile
COPY server/templates/ /app/templates/
RUN mkdir -p /app/data
VOLUME ["/app/data"]
```

### New go.mod dependencies
```
go.etcd.io/bbolt v1.4.x           # embedded key-value store (local mode)
github.com/go-chi/chi/v5 v5.x     # HTTP router
gorm.io/gorm v1.25.x              # ORM — auto-migration, associations, hooks
gorm.io/driver/mysql v1.5.x       # GORM MySQL driver
gorm.io/datatypes v1.2.x          # JSON column type support
github.com/google/uuid v1.6.x     # theme ID generation
```
All pure Go, CGO_ENABLED=0 compatible.

---

## Implementation Order

```
Phase 1 (Store)           ← foundation, everything depends on this
  ↓
Phase 2 (Modes) + Phase 3 (Auth)  ← can be built in parallel
  ↓
Phase 4 (Router/API)      ← wires store + modes + auth together
  ↓
Phase 5 (UI)              ← templates + JS, depends on API
  ↓
Phase 6 (Wiring)          ← final integration in main.go + Dockerfile
```

---

## Verification Plan

1. **Local mode (bbolt, no auth)**: Start server → board auto-registers via `X-Device-ID` → appears on dashboard → click into board view → change mode/theme → see live preview update
2. **Live draft preview**: Open board view → expand theme customizer → change a route color → board canvas instantly re-renders with draft color
3. **Theme save/load**: Save custom theme → select it on another board → verify pixel output matches
4. **Auth enforcement**: Set `ENFORCE_AUTH=true` + `ADMIN_EMAIL` → unauthenticated requests get 401 → admin sees all boards → regular user sees only assigned boards
5. **Board access**: Admin grants user access to a board → user can now see that board on their dashboard and open `/boards/{mac}`
6. **MySQL mode**: Set `MYSQL_DSN` → tables auto-create → all operations work identically to bbolt
7. **Firmware compatibility**: ESP32 continues polling `/api/v1/pixels` with `X-Device-ID` header → same protobuf response → zero firmware changes needed

---

## Critical Files

### To modify
- `server/internal/api/pixels.go` — per-device rendering via mode system
- `server/internal/api/server.go` — chi router, middleware chains, dependency injection
- `server/cmd/subway-server/main.go` — wire store, modes, auth, templates
- `server/Dockerfile` — templates dir, data volume
- `server/go.mod` — new dependencies

### To create
- `server/internal/model/model.go` — domain types
- `server/internal/store/` — store interface + bbolt + mysql backends + factory + seed
- `server/internal/mode/` — mode interface + track + snake implementations
- `server/internal/middleware/` — auth + authz + context
- `server/internal/ui/renderer.go` — template rendering
- `server/templates/` — all HTML templates
- `server/static/preview.js` — client-side draft theme preview

### Unchanged
- `server/internal/mta/` — aggregator, feeds, gtfsrt (no changes needed)
- `server/gen/subwaypb/` — protobuf types
- `server/static/board.js` — reused as-is for canvas rendering
- `server/static/board.svg`, `leds.json` — reused as-is
- All firmware code — zero changes
