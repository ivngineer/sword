package installed

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

// DesktopEntry is a parsed .desktop file owned by an installed package.
type DesktopEntry struct {
	Path string // absolute path to the .desktop file
	Icon string // raw Icon= value (may be a name or absolute path), "" if absent
}

// Desktops maps installed pacman/AUR package names to their user-facing
// .desktop entries. Entries with `NoDisplay=true` or `Hidden=true` are dropped
// during the scan — they're services, KCMs and helper agents the system menu
// also hides, so they shouldn't appear on the Installed screen either. A
// package whose only entries are hidden drops out entirely.
type Desktops map[string][]DesktopEntry

// Owns reports whether pkg owns at least one user-facing .desktop file.
func (d Desktops) Owns(pkg string) bool {
	_, ok := d[pkg]
	return ok
}

// ScanDesktops runs a single `pacman -Ql` to find .desktop files owned by
// installed packages, then parses each to drop hidden entries and capture
// the Icon= value. Returns an empty map when pacman is missing.
func ScanDesktops(ctx context.Context) Desktops {
	out := Desktops{}
	if _, err := exec.LookPath("pacman"); err != nil {
		return out
	}
	raw, err := exec.CommandContext(ctx, "pacman", "-Ql").Output()
	if err != nil {
		return out
	}
	prefixes := [][]byte{
		[]byte("/usr/share/applications/"),
		[]byte("/usr/local/share/applications/"),
	}
	suffix := []byte(".desktop")
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		sp := bytes.IndexByte(line, ' ')
		if sp <= 0 || sp+1 >= len(line) {
			continue
		}
		path := line[sp+1:]
		if !bytes.HasSuffix(path, suffix) {
			continue
		}
		ok := false
		for _, p := range prefixes {
			if bytes.HasPrefix(path, p) {
				ok = true
				break
			}
		}
		if !ok {
			continue
		}
		pkg := string(line[:sp])
		pathStr := string(path)
		entry, visible := parseDesktop(pathStr)
		if !visible {
			continue
		}
		entry.Path = pathStr
		out[pkg] = append(out[pkg], entry)
	}
	return out
}

// parseDesktop reads the first [Desktop Entry] section and returns the icon
// name plus whether the entry is user-visible (NoDisplay/Hidden both false).
func parseDesktop(path string) (DesktopEntry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return DesktopEntry{}, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 4096), 64*1024)
	var entry DesktopEntry
	visible := true
	inEntry := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if inEntry {
				break // hit next section, done with [Desktop Entry]
			}
			inEntry = line == "[Desktop Entry]"
			continue
		}
		if !inEntry || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "Icon="):
			entry.Icon = strings.TrimSpace(strings.TrimPrefix(line, "Icon="))
		case strings.HasPrefix(line, "NoDisplay="):
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "NoDisplay=")), "true") {
				visible = false
			}
		case strings.HasPrefix(line, "Hidden="):
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "Hidden=")), "true") {
				visible = false
			}
		}
	}
	return entry, visible
}
