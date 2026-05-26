package registry

import (
	"strings"

	"sword/backend/installed"
	"sword/backend/metadata"
	"sword/backend/models"
)

// NormalizeName lowercases, strips trademark symbols, and collapses whitespace.
func NormalizeName(s string) string { return normalizeName(s) }

func normalizeName(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("™", "", "®", "", "©", "").Replace(s)
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// DedupKey returns the canonical grouping key for a package: its AppStream id
// when available, otherwise its normalized display name.
func DedupKey(p models.SourcePackage) string {
	if p.AppStreamID != "" {
		return "as:" + strings.ToLower(p.AppStreamID)
	}
	return "nm:" + normalizeName(p.DisplayName)
}

// authority ranks sources for picking canonical metadata: AppStream feeds win
// over everything (applied separately), then pacman > flatpak > aur.
func authority(name string) int {
	switch name {
	case "pacman":
		return 3
	case "flatpak":
		return 2
	case "aur":
		return 1
	}
	return 0
}

// sourceRank mirrors the frontend's source ranking (pacman best). The
// best-ranked source in an entry is marked recommended.
func sourceRank(t string) int {
	switch t {
	case "pacman":
		return 0
	case "aur":
		return 1
	case "flatpak":
		return 2
	}
	return 9
}

// Merge combines packages that belong to the same application into one
// AppEntry. All sources are preserved; metadata is taken from the most
// authoritative source, with AppStream records overriding everything.
func Merge(pkgs []models.SourcePackage, resolvers []metadata.AppStreamResolver) *models.AppEntry {
	if len(pkgs) == 0 {
		return nil
	}
	e := &models.AppEntry{}
	bestAuth := -1
	var appStreamID, pkgName string
	for _, p := range pkgs {
		if appStreamID == "" && p.AppStreamID != "" {
			appStreamID = p.AppStreamID
		}
		if pkgName == "" && p.SourceName == "pacman" {
			pkgName = p.ID
		}
		e.Sources = append(e.Sources, models.AppSource{
			ID:          p.SourceName + ":" + p.ID,
			Type:        p.SourceName,
			PackageName: p.ID,
			Version:     p.Version,
			SizeBytes:   p.SizeBytes,
		})
		if a := authority(p.SourceName); a > bestAuth {
			bestAuth = a
			e.Name = p.DisplayName
			e.Description = p.Description
			e.Publisher = p.Publisher
		}
	}
	e.AppStreamID = appStreamID

	if rec := metadata.Resolve(resolvers, appStreamID, pkgName, pkgs[0].ID); rec != nil {
		if rec.Name != "" {
			e.Name = rec.Name
		}
		if rec.Summary != "" {
			e.Description = rec.Summary
		}
		if rec.Developer != "" {
			e.Publisher = rec.Developer
		}
		if e.AppStreamID == "" {
			e.AppStreamID = rec.ID
		}
	}
	e.IconURL = metadata.ResolveIcon(resolvers, e.AppStreamID, pkgName, pkgs[0].ID)

	if e.Name == "" {
		e.Name = pkgs[0].DisplayName
	}
	if e.AppStreamID != "" {
		e.ID = strings.ToLower(e.AppStreamID)
	} else {
		e.ID = strings.ToLower(e.Name)
	}
	SetRecommended(e)
	e.Status = models.StatusAvailable
	return e
}

// ApplyInstalled stamps Installed on every matching source and flips the
// entry status to "installed" when any source is installed. Safe to call
// multiple times.
func ApplyInstalled(e *models.AppEntry, snap installed.Snapshot) {
	any := false
	for i := range e.Sources {
		hit := snap.Has(e.Sources[i].Type, e.Sources[i].PackageName)
		e.Sources[i].Installed = hit
		if hit {
			any = true
		}
	}
	if any {
		e.Status = models.StatusInstalled
	} else if e.Status == "" {
		e.Status = models.StatusAvailable
	}
}

// SetRecommended flags exactly one source as recommended, picking the best by
// sourceRank. Call it again after appending sources to an existing entry.
func SetRecommended(e *models.AppEntry) {
	if len(e.Sources) == 0 {
		return
	}
	best := 0
	for i := range e.Sources {
		e.Sources[i].IsRecommended = false
		if sourceRank(e.Sources[i].Type) < sourceRank(e.Sources[best].Type) {
			best = i
		}
	}
	e.Sources[best].IsRecommended = true
}
