// Package registry holds the in-memory application index built from all
// enumerable sources.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sahilm/fuzzy"

	"sword/backend/installed"
	"sword/backend/metadata"
	"sword/backend/models"
	"sword/backend/sources"
)

// LocalPkgQuery returns metadata for installed packages by name. The pacman
// source supplies this; passed in so the registry stays decoupled from any
// specific source package.
type LocalPkgQuery func(ctx context.Context, names []string) ([]models.SourcePackage, error)

// AppIndex is the in-memory, deduplicated catalog. It is safe for concurrent
// use and is rebuilt periodically in the background.
type AppIndex struct {
	mu      sync.RWMutex
	entries map[string]*models.AppEntry
	ordered []*models.AppEntry

	buildMu sync.Mutex // serializes concurrent Build calls
	srcs      []sources.Source
	resolvers []metadata.AppStreamResolver

	installedMu sync.RWMutex
	installed   installed.Snapshot

	popularMu   sync.RWMutex
	popularIDs       []string          // ordered by popularity rank, refreshed with the index
	popularIcons     map[string]string // appID -> icon URL from the popular API response
	popularSummaries map[string]string // appID -> summary from the popular API response
}

// NewAppIndex returns an empty index. Pass only enumerable sources (pacman,
// flatpak); the AUR cannot be listed and is queried live by the search layer.
func NewAppIndex(srcs []sources.Source, resolvers []metadata.AppStreamResolver) *AppIndex {
	return &AppIndex{
		entries:   map[string]*models.AppEntry{},
		srcs:      srcs,
		resolvers: resolvers,
		installed: installed.Empty(),
	}
}

// Installed returns the latest installed snapshot. Safe for concurrent use.
func (ix *AppIndex) Installed() installed.Snapshot {
	ix.installedMu.RLock()
	defer ix.installedMu.RUnlock()
	return ix.installed
}

// RefreshInstalled re-queries the installed snapshot and re-stamps every
// entry's per-source installed flag + overall status. Much cheaper than a
// full Build — useful right after an install/remove action so the next Get
// returns the new state without waiting for the heavier rebuild.
func (ix *AppIndex) RefreshInstalled(ctx context.Context) {
	snap := installed.Load(ctx)
	ix.installedMu.Lock()
	ix.installed = snap
	ix.installedMu.Unlock()

	ix.mu.Lock()
	for _, e := range ix.ordered {
		ApplyInstalled(e, snap)
	}
	ix.mu.Unlock()
}

// Build queries every source, enriches packages with AppStream ids, merges
// duplicates and atomically swaps in the new index. A failing source is
// logged and skipped.
func (ix *AppIndex) Build(ctx context.Context) {
	ix.buildMu.Lock()
	defer ix.buildMu.Unlock()

	snap := installed.Load(ctx)
	ix.installedMu.Lock()
	ix.installed = snap
	ix.installedMu.Unlock()

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
	groupOrder := make([]string, 0)
	for _, p := range all {
		k := DedupKey(p)
		if _, exists := groups[k]; !exists {
			groupOrder = append(groupOrder, k)
		}
		groups[k] = append(groups[k], p)
	}

	// Second-pass fold: groups sharing a normalized display name collapse
	// into one. AppStream-keyed groups win as the canonical bucket so that
	// e.g. pacman "steam" folds into the flatpak "Steam" entry instead of
	// remaining a separate card.
	byName := map[string]string{}            // normalized name -> canonical group key
	folded := map[string][]models.SourcePackage{}
	foldedOrder := make([]string, 0, len(groupOrder))
	for _, k := range groupOrder {
		g := groups[k]
		nameKey := ""
		for _, p := range g {
			if n := normalizeName(p.DisplayName); n != "" {
				nameKey = n
				break
			}
		}
		isAS := strings.HasPrefix(k, "as:")
		if nameKey != "" {
			if canon, ok := byName[nameKey]; ok {
				folded[canon] = append(folded[canon], g...)
				if isAS && !strings.HasPrefix(canon, "as:") {
					folded[k] = folded[canon]
					delete(folded, canon)
					for i, kk := range foldedOrder {
						if kk == canon {
							foldedOrder[i] = k
							break
						}
					}
					byName[nameKey] = k
				}
				continue
			}
			byName[nameKey] = k
		}
		folded[k] = append(folded[k], g...)
		foldedOrder = append(foldedOrder, k)
	}

	entries := map[string]*models.AppEntry{}
	ordered := make([]*models.AppEntry, 0, len(foldedOrder))
	for _, k := range foldedOrder {
		e := Merge(folded[k], ix.resolvers)
		if e == nil {
			continue
		}
		ApplyInstalled(e, snap)
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

// InstalledApps returns every installed application as a flat, alphabetically
// sorted list. Pacman-native and Flatpak entries come from the index (so they
// carry resolved icons and merged metadata). AUR-foreign packages absent from
// the index are filled in via localPacQuery on the local pacman DB. Pass nil
// for localPacQuery to skip AUR augmentation.
func (ix *AppIndex) InstalledApps(ctx context.Context, localPacQuery LocalPkgQuery) []models.AppEntry {
	// Only packages that ship a desktop entry — drops libs, daemons, helpers.
	// Flatpaks bypass the filter (every flatpak app is user-facing by design).
	// The same scan also feeds icon resolution: parsing each .desktop's Icon=
	// field is the only way to find icons for apps whose desktop file / icon
	// name doesn't match their package name (e.g. gnome-terminal → org.gnome.Terminal).
	desktops := installed.ScanDesktops(ctx)
	desktopIcons := installed.ResolveIcons(desktops)

	ix.mu.RLock()
	out := make([]models.AppEntry, 0)
	covered := map[string]struct{}{} // "type:packageName" already in result
	for _, e := range ix.ordered {
		if e.Status != models.StatusInstalled {
			continue
		}
		if !entryIsUserFacing(e, desktops) {
			continue
		}
		cp := *e
		out = append(out, cp)
		for _, s := range e.Sources {
			if s.Installed {
				covered[s.Type+":"+s.PackageName] = struct{}{}
			}
		}
	}
	ix.mu.RUnlock()

	// Icon/summary fallback from the popular cache, matching Get/Popular.
	ix.popularMu.RLock()
	icons := ix.popularIcons
	summaries := ix.popularSummaries
	ix.popularMu.RUnlock()
	for i := range out {
		if out[i].IconURL == "" {
			out[i].IconURL = icons[out[i].ID]
		}
		if out[i].IconURL == "" {
			// Last resort: walk the entry's pacman/AUR sources and try the
			// .desktop-derived icon map. Catches everything not in AppStream.
			for _, s := range out[i].Sources {
				if !s.Installed || s.Type == "flatpak" {
					continue
				}
				if u := desktopIcons[s.PackageName]; u != "" {
					out[i].IconURL = u
					break
				}
			}
		}
		if out[i].Description == "" {
			out[i].Description = summaries[out[i].ID]
		}
	}

	// AUR-only: foreign pacman names not already represented.
	snap := ix.Installed()
	missing := make([]string, 0, len(snap.AUR))
	for name := range snap.AUR {
		if _, ok := covered["aur:"+name]; ok {
			continue
		}
		if _, ok := covered["pacman:"+name]; ok {
			continue
		}
		if !desktops.Owns(name) {
			continue
		}
		missing = append(missing, name)
	}
	if len(missing) > 0 && localPacQuery != nil {
		pkgs, err := localPacQuery(ctx, missing)
		if err != nil {
			log.Printf("registry: installed AUR query: %v", err)
		}
		for _, p := range pkgs {
			e := models.AppEntry{
				ID:          strings.ToLower(p.ID),
				Name:        p.DisplayName,
				Description: p.Description,
				Status:      models.StatusInstalled,
				Sources: []models.AppSource{{
					ID:            "aur:" + p.ID,
					Type:          "aur",
					PackageName:   p.ID,
					Version:       p.Version,
					SizeBytes:     p.SizeBytes,
					IsRecommended: true,
					Installed:     true,
				}},
			}
			if rec := metadata.Resolve(ix.resolvers, p.ID); rec != nil {
				if rec.Name != "" {
					e.Name = rec.Name
				}
				if rec.Summary != "" {
					e.Description = rec.Summary
				}
				if rec.Developer != "" {
					e.Publisher = rec.Developer
				}
				if rec.ID != "" {
					e.AppStreamID = rec.ID
					e.ID = strings.ToLower(rec.ID)
				}
				if len(rec.Screenshots) > 0 {
					e.Screenshots = append([]string(nil), rec.Screenshots...)
				}
			}
			e.IconURL = metadata.ResolveIcon(ix.resolvers, e.AppStreamID, p.ID)
			if e.IconURL == "" {
				e.IconURL = desktopIcons[p.ID]
			}
			out = append(out, e)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
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

// Get returns a copy of the entry with the given canonical id. Falls back to
// the Flathub popular-list icon/summary when the index entry lacks them, so
// detail screens get the same enriched fields as the popular grid.
func (ix *AppIndex) Get(id string) (*models.AppEntry, error) {
	key := strings.ToLower(id)

	ix.mu.RLock()
	e, ok := ix.entries[key]
	if !ok {
		ix.mu.RUnlock()
		return nil, fmt.Errorf("app not found: %s", id)
	}
	cp := *e
	ix.mu.RUnlock()

	ix.popularMu.RLock()
	if cp.IconURL == "" {
		cp.IconURL = ix.popularIcons[key]
	}
	if cp.Description == "" {
		cp.Description = ix.popularSummaries[key]
	}
	ix.popularMu.RUnlock()

	return &cp, nil
}

// Popular returns apps from the index ordered by Flathub popularity rank.
// Apps not present in the index are skipped. Returns nil before the first
// successful fetch.
func (ix *AppIndex) Popular() []models.AppEntry {
	ix.popularMu.RLock()
	ids := append([]string(nil), ix.popularIDs...)
	icons := ix.popularIcons
	summaries := ix.popularSummaries
	ix.popularMu.RUnlock()

	ix.mu.RLock()
	defer ix.mu.RUnlock()

	out := make([]models.AppEntry, 0, len(ids))
	for _, id := range ids {
		if e, ok := ix.entries[id]; ok {
			cp := *e
			if cp.IconURL == "" {
				cp.IconURL = icons[id]
			}
			if cp.Description == "" {
				cp.Description = summaries[id]
			}
			out = append(out, cp)
		}
	}
	return out
}

// refreshPopular fetches the Flathub popular-last-month list and caches the
// top 20 app IDs. Called from Build so it refreshes with the index.
func (ix *AppIndex) refreshPopular() {
	const url = "https://flathub.org/api/v2/collection/popular"
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		log.Printf("registry: popular fetch: %v", err)
		return
	}
	defer resp.Body.Close()

	var payload struct {
		Hits []struct {
			AppID   string `json:"app_id"`
			Icon    string `json:"icon"`
			Summary string `json:"summary"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		log.Printf("registry: popular decode: %v", err)
		return
	}

	const limit = 20
	ids := make([]string, 0, limit)
	icons := make(map[string]string, limit)
	summaries := make(map[string]string, limit)
	for _, h := range payload.Hits {
		if len(ids) >= limit {
			break
		}
		if h.AppID != "" {
			key := strings.ToLower(h.AppID)
			ids = append(ids, key)
			if h.Icon != "" {
				icons[key] = h.Icon
			}
			if h.Summary != "" {
				summaries[key] = h.Summary
			}
		}
	}

	ix.popularMu.Lock()
	ix.popularIDs = ids
	ix.popularIcons = icons
	ix.popularSummaries = summaries
	ix.popularMu.Unlock()
	log.Printf("registry: popular list cached, %d ids", len(ids))
}

// entryIsUserFacing reports whether an installed entry should appear on the
// Installed screen. Flatpaks always pass. Pacman/AUR entries must own a
// .desktop file (signalling a GUI or terminal-launchable app rather than a
// library/daemon).
func entryIsUserFacing(e *models.AppEntry, desktops installed.Desktops) bool {
	for _, s := range e.Sources {
		if !s.Installed {
			continue
		}
		if s.Type == "flatpak" {
			return true
		}
		if desktops.Owns(s.PackageName) {
			return true
		}
	}
	return false
}
