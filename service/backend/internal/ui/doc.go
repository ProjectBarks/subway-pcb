// Package ui contains templ components for server-rendered HTML.
//
// File categories:
//
// Pages (wrapped in Base layout):
//   - dashboard.templ — device dashboard
//   - board.templ     — individual board view
//   - community.templ — community plugin browser
//   - editor.templ    — Lua plugin editor
//
// Standalone pages (own layout):
//   - landing.templ   — public landing page
//   - login.templ     — OAuth login page
//
// Shared components:
//   - nav.templ       — navigation bar
//   - icon.templ      — SVG icon helpers
//   - toast.templ     — toast notifications
//
// Page partials (HTMX fragments):
//   - controls.templ  — board control panel
//   - access.templ    — board access/sharing
//
// Layout:
//   - layout.templ    — Base HTML wrapper, head, scripts
//
// Go support:
//   - types.go        — shared view model types
//   - helpers.go      — template helper functions
package ui
