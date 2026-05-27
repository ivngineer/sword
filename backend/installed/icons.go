package installed

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ResolveIcons maps each package in d to a `file://` URL pointing at an
// on-disk icon resolved through its .desktop entries' `Icon=` fields. Packages
// with no resolvable icon are omitted (caller can fall back).
//
// Strategy:
//  1. Build an in-memory index of every icon file under standard theme dirs
//     (one walk, not one stat-per-package).
//  2. For each desktop file, parse the `Icon=` line.
//  3. If absolute, use directly. Otherwise look up the name in the index and
//     pick the largest available size.
func ResolveIcons(d Desktops) map[string]string {
	if len(d) == 0 {
		return map[string]string{}
	}
	idx := buildIconIndex()
	out := map[string]string{}
	for pkg, entries := range d {
		for _, e := range entries {
			name := e.Icon
			if name == "" {
				continue
			}
			if filepath.IsAbs(name) {
				if fileExists(name) {
					out[pkg] = "file://" + name
					break
				}
				continue
			}
			if res := idx.best(name); res != "" {
				out[pkg] = "file://" + res
				break
			}
		}
	}
	return out
}

// iconIndex maps a stem name ("firefox") to candidate on-disk icon files,
// ranked so best().score wins for the largest raster / scalable svg.
type iconIndex struct {
	byName map[string][]iconHit
}

type iconHit struct {
	path  string
	score int // higher = preferred
}

func (ix *iconIndex) best(name string) string {
	// Try the icon name verbatim first, then stripped of an extension —
	// some desktop files write `Icon=foo.png` even though the spec says no.
	candidates := []string{name}
	if ext := filepath.Ext(name); ext != "" {
		candidates = append(candidates, strings.TrimSuffix(name, ext))
	}
	for _, c := range candidates {
		hits := ix.byName[strings.ToLower(c)]
		if len(hits) == 0 {
			continue
		}
		best := hits[0]
		for _, h := range hits[1:] {
			if h.score > best.score {
				best = h
			}
		}
		return best.path
	}
	return ""
}

// buildIconIndex walks standard icon theme + pixmap dirs once and groups every
// icon file by stem name. Handles both layouts seen in the wild:
//
//	<theme>/<size>/apps/<name>.<ext>   (hicolor, Adwaita, Papirus, ...)
//	<theme>/apps/<size>/<name>.<ext>   (breeze, oxygen, ...)
//
// Files under any /apps/ segment are kept; size is inferred from whichever
// path segment parses as "NxN" or equals "scalable".
func buildIconIndex() *iconIndex {
	ix := &iconIndex{byName: map[string][]iconHit{}}
	for _, root := range []string{
		"/usr/share/icons",
		"/usr/local/share/icons",
		"/var/lib/flatpak/exports/share/icons",
	} {
		themes, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, th := range themes {
			if !th.IsDir() {
				continue
			}
			walkThemeIcons(ix, filepath.Join(root, th.Name()), th.Name())
		}
	}
	for _, pix := range []string{"/usr/share/pixmaps", "/usr/local/share/pixmaps"} {
		addPixmaps(ix, pix)
	}
	return ix
}

func walkThemeIcons(ix *iconIndex, themeDir, themeName string) {
	_ = filepath.WalkDir(themeDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".png" && ext != ".svg" && ext != ".xpm" {
			return nil
		}
		// Walk every category dir, not just apps/. The freedesktop icon spec
		// resolves an `Icon=name` lookup across all categories (actions,
		// places, preferences, mimetypes, ...) so we must too — many KDE/GTK
		// apps reference icons that live outside apps/.
		rel := strings.TrimPrefix(path, themeDir)
		segs := strings.Split(rel, string(filepath.Separator))
		size := ""
		isApps := false
		for _, s := range segs {
			if s == "apps" {
				isApps = true
			}
			if s == "scalable" || parseSize(s) > 0 {
				size = s
			}
		}
		stem := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		score := iconScore(size, ext, themeName)
		if isApps {
			score += 100 // prefer the dedicated apps/ entry when both exist
		}
		ix.byName[stem] = append(ix.byName[stem], iconHit{
			path:  path,
			score: score,
		})
		return nil
	})
}

func addPixmaps(ix *iconIndex, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".png" && ext != ".svg" && ext != ".xpm" {
			continue
		}
		stem := strings.ToLower(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		ix.byName[stem] = append(ix.byName[stem], iconHit{
			path:  filepath.Join(dir, e.Name()),
			score: iconScore("pixmaps", ext, ""),
		})
	}
}

// iconScore ranks candidates so the picker prefers hicolor scalable svgs and
// large rasters. Tuned for "looks good at 64-128px" — what the UI shows.
func iconScore(sizeLabel, ext, theme string) int {
	score := 0
	if theme == "hicolor" {
		score += 10
	}
	if sizeLabel == "scalable" || ext == ".svg" {
		score += 1000
	} else if sizeLabel == "pixmaps" {
		score += 50
	} else if n := parseSize(sizeLabel); n > 0 {
		score += 500 - absInt(128-n)
	}
	if ext == ".png" {
		score += 5
	}
	return score
}

func parseSize(label string) int {
	if i := strings.Index(label, "x"); i > 0 {
		n, err := strconv.Atoi(label[:i])
		if err == nil {
			return n
		}
	}
	return 0
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
