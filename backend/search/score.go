package search

import (
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"

	"sword/backend/models"
)

// sourceWeight gives a small ranking nudge by the best source backing an entry.
func sourceWeight(e models.AppEntry) int {
	w := 0
	for _, s := range e.Sources {
		switch s.Type {
		case "pacman":
			if w < 3 {
				w = 3
			}
		case "flatpak":
			if w < 2 {
				w = 2
			}
		case "aur":
			if w < 1 {
				w = 1
			}
		}
	}
	return w
}

// nameBucket classifies how well an entry name matches the query. The bucket
// dominates the final score so that a substring match always outranks a
// fuzzy-only match — raw fuzzy scores are large and would otherwise let an
// app that merely mentions the query in its name beat an exact hit.
//
// Returns the bucket value, or -1 when the name does not match at all.
func nameBucket(q, name string) int {
	switch {
	case name == q:
		return 10000
	case strings.HasPrefix(name, q):
		return 6000
	case strings.Contains(name, q):
		return 3000
	}
	m := fuzzy.Find(q, []string{name})
	if len(m) == 0 || m[0].Score <= 0 {
		return -1
	}
	s := m[0].Score
	if s > 1000 {
		s = 1000
	}
	return s
}

// metaBonus is a small, bounded tiebreaker for matches in publisher or
// description text.
func metaBonus(q string, e models.AppEntry) int {
	b := 0
	if q != "" && strings.Contains(strings.ToLower(e.Publisher), q) {
		b += 30
	}
	if q != "" && strings.Contains(strings.ToLower(e.Description), q) {
		b += 20
	}
	return b
}

// Score computes a composite relevance score for each entry — name match
// bucket (dominant), with fuzzy publisher/description matches and source
// weight as bounded tiebreakers — and returns the entries sorted best-first.
// Entries that match nothing are dropped.
func Score(query string, entries []models.AppEntry) []models.IndexEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]models.IndexEntry, 0, len(entries))
	for _, e := range entries {
		bucket := nameBucket(q, strings.ToLower(e.Name))
		meta := metaBonus(q, e)
		if bucket < 0 {
			if meta == 0 {
				continue // matches neither name nor metadata
			}
			bucket = 500 // metadata-only match
		}
		// Tiebreaker stays small relative to the bucket gap (×100).
		// guiBoost floats apps with AppStream records (GUI/desktop apps) above
		// CLI tools and libraries when name-match quality is equal.
		guiBoost := 0
		if e.AppStreamID != "" {
			guiBoost = 200
		}
		tb := sourceWeight(e) + meta - len(e.Name) + guiBoost
		out = append(out, models.IndexEntry{App: e, Score: bucket*100 + tb})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}
