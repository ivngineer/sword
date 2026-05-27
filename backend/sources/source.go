// Package sources defines the Source extensibility contract. Adding a new
// package source means one new file implementing Source plus one registration
// line in main.go.
package sources

import (
	"context"

	"sword/backend/models"
)

// ProgressFn receives parsed progress updates from a long-running action.
// fraction is in [0,1] when known, or negative for indeterminate ticks.
// status is a short human-readable description of the current step.
type ProgressFn func(fraction float64, status string)

// NopProgress is a ProgressFn that discards updates. Use when a caller does
// not care about progress.
func NopProgress(float64, string) {}

// Source is implemented by every package source (pacman, aur, flatpak, ...).
type Source interface {
	// Name returns the stable source identifier ("pacman", "aur", "flatpak").
	Name() string
	// Search returns packages matching query. An empty query means "all"
	// for enumerable sources; non-enumerable sources may return nil.
	Search(ctx context.Context, query string) ([]models.SourcePackage, error)
	// Get returns a single package by its source-local id.
	Get(ctx context.Context, id string) (models.SourcePackage, error)
	// Install installs a package by its source-local id. onProgress is
	// called as the underlying tool reports progress; never nil — pass
	// NopProgress when uninterested.
	Install(ctx context.Context, id string, onProgress ProgressFn) error
	// Remove uninstalls a package by its source-local id. onProgress is
	// called as the underlying tool reports progress; never nil.
	Remove(ctx context.Context, id string, onProgress ProgressFn) error
}
