// Package registry holds the in-memory application index built from all
// enumerable sources.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sahilm/fuzzy"

	"sword/backend/metadata"
	"sword/backend/models"
	"sword/backend/sources"
)

// AppIndex is the in-memory, deduplicated catalog. It is safe for concurrent
// use and is rebuilt periodically in the background.
type AppIndex struct {
	mu      sync.RWMutex
	entries map[string]*models.AppEntry
	ordered []*models.AppEntry

	buildMu sync.Mutex // serializes concurrent Build calls
	srcs      []sources.Source
	resolvers []metadata.AppStreamResolver

	popularMu  sync.RWMutex
	popularIDs []string // ordered by popularity rank, refreshed with the index
}

// NewAppIndex returns an empty index. Pass only enumerable sources (pacman,
// flatpak); the AUR cannot be listed and is queried live by the search layer.
func NewAppIndex(srcs []sources.Source, resolvers []metadata.AppStreamResolver) *AppIndex {
	return &AppIndex{
		entries:   map[string]*models.AppEntry{},
		srcs:      srcs,
		resolvers: resolvers,
	}
}

// Build queries every source, enriches packages with AppStream ids, merges
// duplicates and atomically swaps in the new index. A failing source is
// logged and skipped.
func (ix *AppIndex) Build(ctx context.Context) {
	ix.buildMu.Lock()
	defer ix.buildMu.Unlock()

	var all []models.SourcePackage
	for _, s := range ix.srcs {
		pkgs, err := s.Search(ctx, "")
		if err != nil {
			log.Printf("registry: build %s: %v", s.Name(), err)
			continue
		}
		all = append(all, pkgs...)
	}

	// Enrich pacman-style packages with an AppStream id so they dedup against
	// their flatpak counterparts.
	for i := range all {
		if all[i].AppStreamID == "" {
			if rec := metadata.Resolve(ix.resolvers, all[i].ID); rec != nil {
				all[i].AppStreamID = rec.ID
			}
		}
	}

	groups := map[string][]models.SourcePackage{}
	for _, p := range all {
		k := DedupKey(p)
		groups[k] = append(groups[k], p)
	}

	entries := map[string]*models.AppEntry{}
	ordered := make([]*models.AppEntry, 0, len(groups))
	for _, g := range groups {
		e := Merge(g, ix.resolvers)
		if e == nil {
			continue
		}
		entries[e.ID] = e
		ordered = append(ordered, e)
	}

	ix.mu.Lock()
	ix.entries = entries
	ix.ordered = ordered
	ix.mu.Unlock()
	log.Printf("registry: index built, %d entries", len(ordered))

	go ix.refreshPopular()
}

// StartAutoRebuild rebuilds the index every interval until ctx is cancelled.
func (ix *AppIndex) StartAutoRebuild(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				ix.Build(ctx)
			}
		}
	}()
}

type nameSource struct{ list []*models.AppEntry }

func (n nameSource) String(i int) string { return n.list[i].Name }
func (n nameSource) Len() int            { return len(n.list) }

// Search returns index entries whose name fuzzy-matches query, ordered by
// match quality. An empty query returns nil.
func (ix *AppIndex) Search(query string) []models.IndexEntry {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	ix.mu.RLock()
	list := ix.ordered
	ix.mu.RUnlock()

	matches := fuzzy.FindFrom(query, nameSource{list})
	out := make([]models.IndexEntry, 0, len(matches))
	for _, m := range matches {
		out = append(out, models.IndexEntry{App: *list[m.Index], Score: m.Score})
	}
	return out
}

// Get returns a copy of the entry with the given canonical id.
func (ix *AppIndex) Get(id string) (*models.AppEntry, error) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	e, ok := ix.entries[strings.ToLower(id)]
	if !ok {
		return nil, fmt.Errorf("app not found: %s", id)
	}
	cp := *e
	return &cp, nil
}

// Popular returns apps from the index ordered by Flathub popularity rank.
// Apps not present in the index are skipped. Returns nil before the first
// successful fetch.
func (ix *AppIndex) Popular() []models.AppEntry {
	ix.popularMu.RLock()
	ids := append([]string(nil), ix.popularIDs...)
	ix.popularMu.RUnlock()

	ix.mu.RLock()
	defer ix.mu.RUnlock()

	out := make([]models.AppEntry, 0, len(ids))
	for _, id := range ids {
		if e, ok := ix.entries[id]; ok {
			out = append(out, *e)
		}
	}
	return out
}

// refreshPopular fetches the Flathub popular-last-month list and caches the
// top 20 app IDs. Called from Build so it refreshes with the index.
func (ix *AppIndex) refreshPopular() {
	const url = "https://flathub.org/api/v2/popular/last-month"
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		log.Printf("registry: popular fetch: %v", err)
		return
	}
	defer resp.Body.Close()

	var payload struct {
		Hits []struct {
			AppID string `json:"app_id"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		log.Printf("registry: popular decode: %v", err)
		return
	}

	const limit = 20
	ids := make([]string, 0, limit)
	for _, h := range payload.Hits {
		if len(ids) >= limit {
			break
		}
		if h.AppID != "" {
			ids = append(ids, strings.ToLower(h.AppID))
		}
	}

	ix.popularMu.Lock()
	ix.popularIDs = ids
	ix.popularMu.Unlock()
	log.Printf("registry: popular list cached, %d ids", len(ids))
}
