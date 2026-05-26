// Package installed reports which packages are currently installed on the
// system, partitioned by source. Snapshots are cheap shell calls; callers
// refresh on the same cadence as the registry index.
package installed

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

// Snapshot holds installed-package sets keyed by source-local package name.
// All maps are non-nil; absence means "not installed".
type Snapshot struct {
	Pacman  map[string]struct{} // native sync-repo packages (pacman -Qqn)
	AUR     map[string]struct{} // foreign packages (pacman -Qqm)
	Flatpak map[string]struct{} // flatpak application ids
}

// Empty returns a usable, empty snapshot.
func Empty() Snapshot {
	return Snapshot{
		Pacman:  map[string]struct{}{},
		AUR:     map[string]struct{}{},
		Flatpak: map[string]struct{}{},
	}
}

// Load queries the system for installed packages. Missing tools degrade to
// empty sets — the snapshot is always usable.
func Load(ctx context.Context) Snapshot {
	s := Empty()
	if _, err := exec.LookPath("pacman"); err == nil {
		s.Pacman = runSet(ctx, "pacman", "-Qqn")
		s.AUR = runSet(ctx, "pacman", "-Qqm")
	}
	if _, err := exec.LookPath("flatpak"); err == nil {
		s.Flatpak = runSet(ctx, "flatpak", "list", "--app", "--columns=application")
	}
	return s
}

// HasPacman reports whether name is in the native pacman set.
func (s Snapshot) HasPacman(name string) bool { _, ok := s.Pacman[name]; return ok }

// HasAUR reports whether name is in the foreign (AUR) set.
func (s Snapshot) HasAUR(name string) bool { _, ok := s.AUR[name]; return ok }

// HasFlatpak reports whether the flatpak application id is installed.
func (s Snapshot) HasFlatpak(id string) bool { _, ok := s.Flatpak[id]; return ok }

// Has reports whether a single (sourceType, packageName) pair is installed.
func (s Snapshot) Has(sourceType, name string) bool {
	switch sourceType {
	case "pacman":
		return s.HasPacman(name)
	case "aur":
		return s.HasAUR(name)
	case "flatpak":
		return s.HasFlatpak(name)
	}
	return false
}

func runSet(ctx context.Context, name string, args ...string) map[string]struct{} {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return map[string]struct{}{}
	}
	set := map[string]struct{}{}
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		v := strings.TrimSpace(string(line))
		if v != "" {
			set[v] = struct{}{}
		}
	}
	return set
}
