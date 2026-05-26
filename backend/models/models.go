// Package models holds the shared data types exchanged between sources,
// the registry, the search layer and the IPC protocol.
package models

// SourcePackage is one package as reported by a single source. It is the raw,
// pre-merge representation. JSON tags are camelCase to match the frontend.
type SourcePackage struct {
	SourceName  string `json:"sourceName"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Publisher   string `json:"publisher"`
	Version     string `json:"version"`
	Description string `json:"description"`
	License     string `json:"license"`
	SizeBytes   int64  `json:"sizeBytes"`
	AppStreamID string `json:"appStreamId"`
}

// AppSource is the per-source view rendered inside an app card. Field names
// match the frontend AppSource type exactly.
type AppSource struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	PackageName   string `json:"packageName"`
	Version       string `json:"version"`
	SizeBytes     int64  `json:"sizeBytes"`
	IsRecommended bool   `json:"isRecommended"`
	Installed     bool   `json:"installed"`
}

// Status values for AppEntry.Status.
const (
	StatusAvailable = "available"
	StatusInstalled = "installed"
)

// AppEntry is a deduplicated application, possibly backed by several sources.
// Field names match the frontend AppEntry type exactly.
type AppEntry struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Publisher   string      `json:"publisher"`
	Description string      `json:"description"`
	IconURL     string      `json:"iconUrl"`
	Status      string      `json:"status"`
	Sources     []AppSource `json:"sources"`
	Screenshots []string    `json:"screenshots,omitempty"`
	// AppStreamID is internal bookkeeping for dedup/icon resolution; the
	// frontend ignores it.
	AppStreamID string `json:"appStreamId,omitempty"`
}

// IndexEntry pairs an AppEntry with its relevance score for a query.
type IndexEntry struct {
	App   AppEntry `json:"app"`
	Score int      `json:"score"`
}

// SourceInfo reports whether a given source is usable on this system.
type SourceInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}
