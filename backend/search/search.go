// Package search orchestrates two-phase search: a fast in-memory local phase
// followed by a network phase that merges live AUR results.
package search

import (
	"context"
	"log"
	"strings"

	"sword/backend/metadata"
	"sword/backend/models"
	"sword/backend/registry"
	"sword/backend/sources"
)

// maxResults caps how many entries either phase returns.
const maxResults = 60

// Orchestrator fans search requests out to the registry and the AUR.
type Orchestrator struct {
	index     *registry.AppIndex
	aur       sources.Source
	resolvers []metadata.AppStreamResolver
}

// NewOrchestrator wires the orchestrator to the index and the AUR source.
func NewOrchestrator(index *registry.AppIndex, aur sources.Source, resolvers []metadata.AppStreamResolver) *Orchestrator {
	return &Orchestrator{index: index, aur: aur, resolvers: resolvers}
}

// Local returns the fast phase: pacman + flatpak results from the in-memory
// index, no network.
func (o *Orchestrator) Local(query string) []models.AppEntry {
	apps := limit(indexApps(o.index.Search(query)))
	o.enrichIcons(apps)
	return apps
}

// Complete returns the final phase: local results with live AUR results
// merged in, re-scored. A failed AUR query degrades to local-only results.
func (o *Orchestrator) Complete(ctx context.Context, query string) []models.AppEntry {
	local := indexApps(o.index.Search(query))
	aurPkgs, err := o.aur.Search(ctx, query)
	if err != nil {
		log.Printf("search: aur: %v", err)
	}
	merged := o.mergeAUR(local, aurPkgs)
	apps := limit(scoreApps(Score(query, merged)))
	snap := o.index.Installed()
	for i := range apps {
		registry.ApplyInstalled(&apps[i], snap)
	}
	o.enrichIcons(apps)
	return apps
}

// enrichIcons resolves icons for entries that lack one. The index may be
// built before remote AppStream feeds finish loading, so icon resolution is
// re-tried at query time against the current resolver state.
func (o *Orchestrator) enrichIcons(apps []models.AppEntry) {
	for i := range apps {
		if apps[i].IconURL != "" {
			continue
		}
		keys := make([]string, 0, len(apps[i].Sources)+1)
		keys = append(keys, apps[i].AppStreamID)
		for _, s := range apps[i].Sources {
			keys = append(keys, s.PackageName)
		}
		apps[i].IconURL = metadata.ResolveIcon(o.resolvers, keys...)
	}
}

func (o *Orchestrator) mergeAUR(apps []models.AppEntry, aurPkgs []models.SourcePackage) []models.AppEntry {
	byKey := map[string]int{}
	for i := range apps {
		byKey[entryKey(apps[i])] = i
	}
	for _, p := range aurPkgs {
		k := strings.ToLower(p.DisplayName)
		if i, ok := byKey[k]; ok {
			apps[i].Sources = append(apps[i].Sources, models.AppSource{
				ID:          "aur:" + p.ID,
				Type:        "aur",
				PackageName: p.ID,
				Version:     p.Version,
				SizeBytes:   p.SizeBytes,
			})
			registry.SetRecommended(&apps[i])
			continue
		}
		if e := registry.Merge([]models.SourcePackage{p}, o.resolvers); e != nil {
			apps = append(apps, *e)
			byKey[entryKey(*e)] = len(apps) - 1
		}
	}
	return apps
}

func entryKey(e models.AppEntry) string {
	if e.AppStreamID != "" {
		return strings.ToLower(e.AppStreamID)
	}
	return strings.ToLower(e.Name)
}

func indexApps(idx []models.IndexEntry) []models.AppEntry {
	out := make([]models.AppEntry, 0, len(idx))
	for _, ie := range idx {
		out = append(out, ie.App)
	}
	return out
}

func scoreApps(scored []models.IndexEntry) []models.AppEntry {
	out := make([]models.AppEntry, 0, len(scored))
	for _, ie := range scored {
		out = append(out, ie.App)
	}
	return out
}

func limit(apps []models.AppEntry) []models.AppEntry {
	if len(apps) > maxResults {
		return apps[:maxResults]
	}
	return apps
}
