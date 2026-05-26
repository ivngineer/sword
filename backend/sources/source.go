// Package sources defines the Source extensibility contract. Adding a new
// package source means one new file implementing Source plus one registration
// line in main.go.
package sources

import (
	"context"

	"sword/backend/models"
)

// Source is implemented by every package source (pacman, aur, flatpak, ...).
type Source interface {
	// Name returns the stable source identifier ("pacman", "aur", "flatpak").
	Name() string
	// Search returns packages matching query. An empty query means "all"
	// for enumerable sources; non-enumerable sources may return nil.
	Search(ctx context.Context, query string) ([]models.SourcePackage, error)
	// Get returns a single package by its source-local id.
	Get(ctx context.Context, id string) (models.SourcePackage, error)
	// Install installs a package by its source-local id.
	Install(ctx context.Context, id string) error
	// Remove uninstalls a package by its source-local id.
	Remove(ctx context.Context, id string) error
}
