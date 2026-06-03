// Package drivers scores package-source results to surface driver packages
// (GPU drivers, firmware blobs, DKMS kernel modules, xf86-* / sane-* / v4l,
// etc.) for the Drivers screen. It reuses the existing pacman (expac) and AUR
// sources rather than libalpm: a three-stage pipeline (name regex, description
// keyword scan, dependency check) classifies each package.
package drivers

import (
	"context"
	"log"
	"regexp"
	"strings"

	"sword/backend/installed"
	"sword/backend/models"
	"sword/backend/sources/aur"
	"sword/backend/sources/pacman"
)

// DriverKind values, mirrored by AppEntry.DriverKind. The frontend maps these
// to friendly labels ("Kernel Module", "Firmware", "Driver").
const (
	KindKernelModule = "kernel-module"
	KindFirmware     = "firmware"
	KindDriver       = "driver"
)

// nameRe matches package names that are almost certainly drivers. Single
// compiled pattern, O(n) over the sync DB.
var nameRe = regexp.MustCompile(`(?i)(-dkms$|^xf86-|-firmware$|^linux-firmware|^nvidia|^amdgpu|^broadcom|^v4l|^sane-|^mesa-|^vulkan-|-driver$)`)

// descKeywords promote a near-miss package to a candidate when present in the
// lowercased description.
var descKeywords = []string{"driver", "kernel module", "firmware", "dkms"}

// kmodDeps signal a loadable kernel module rather than pure firmware/userspace.
var kmodDeps = map[string]bool{"dkms": true, "linux-headers": true, "linux-api-headers": true}

// Detector holds the sources and an installed-snapshot accessor. snap is a
// function so it always reflects the index's latest snapshot.
type Detector struct {
	pac  *pacman.Source
	au   *aur.Source
	snap func() installed.Snapshot
}

// NewDetector wires the pacman and AUR sources plus an installed-snapshot
// accessor (typically AppIndex.Installed).
func NewDetector(pac *pacman.Source, au *aur.Source, snap func() installed.Snapshot) *Detector {
	return &Detector{pac: pac, au: au, snap: snap}
}

// candidate reports whether a name/description pair looks like a driver
// (stages 1 and 2). The boolean firmwareHint flags a firmware signal from the
// name/description, used by stage 3 when no kmod dep is present.
func candidate(name, desc string) (isDriver, firmwareHint bool) {
	nameHit := nameRe.MatchString(name)
	low := strings.ToLower(desc)
	descHit := false
	for _, kw := range descKeywords {
		if strings.Contains(low, kw) {
			descHit = true
			break
		}
	}
	if !nameHit && !descHit {
		return false, false
	}
	firmwareHint = strings.HasSuffix(strings.ToLower(name), "-firmware") ||
		strings.HasPrefix(strings.ToLower(name), "linux-firmware") ||
		strings.Contains(low, "firmware")
	return true, firmwareHint
}

// kind resolves the friendly classification. Precedence: kernel-module >
// firmware > driver.
func kind(isKernelModule, isFirmware bool) string {
	switch {
	case isKernelModule:
		return KindKernelModule
	case isFirmware:
		return KindFirmware
	default:
		return KindDriver
	}
}

// Local scores the pacman sync DB. Stage 1/2 over every package, then a single
// batched dependency lookup over the candidates for stage 3.
func (d *Detector) Local(ctx context.Context) []models.AppEntry {
	pkgs, err := d.pac.Search(ctx, "")
	if err != nil {
		log.Printf("drivers: pacman search: %v", err)
		return nil
	}

	type cand struct {
		pkg          models.SourcePackage
		firmwareHint bool
	}
	cands := make([]cand, 0, 64)
	names := make([]string, 0, 64)
	for _, p := range pkgs {
		ok, fw := candidate(p.ID, p.Description)
		if !ok {
			continue
		}
		cands = append(cands, cand{pkg: p, firmwareHint: fw})
		names = append(names, p.ID)
	}

	deps, err := d.pac.Deps(ctx, names)
	if err != nil {
		log.Printf("drivers: pacman deps: %v", err)
		deps = map[string][]string{}
	}

	snap := d.snap()
	out := make([]models.AppEntry, 0, len(cands))
	for _, c := range cands {
		isKmod := false
		for _, dep := range deps[c.pkg.ID] {
			base := depBase(dep)
			if kmodDeps[base] {
				isKmod = true
				break
			}
		}
		out = append(out, entry(c.pkg, "pacman", kind(isKmod, c.firmwareHint), snap.HasPacman(c.pkg.ID)))
	}
	return out
}

// AUR queries the AUR by name for the common driver name shapes and scores the
// results. The search RPC omits dependencies, so the kernel-module flag is
// inferred from the name (a "-dkms" suffix) rather than a real dep check.
func (d *Detector) AUR(ctx context.Context, exclude map[string]bool) []models.AppEntry {
	snap := d.snap()
	seen := map[string]bool{}
	out := make([]models.AppEntry, 0, 32)
	for _, q := range []string{"-dkms", "xf86-", "-firmware"} {
		res, err := d.au.Search(ctx, q)
		if err != nil {
			log.Printf("drivers: aur search %q: %v", q, err)
			continue
		}
		for _, p := range res {
			if exclude[p.ID] || seen[p.ID] {
				continue
			}
			ok, fw := candidate(p.ID, p.Description)
			if !ok {
				continue
			}
			seen[p.ID] = true
			isKmod := strings.HasSuffix(strings.ToLower(p.ID), "-dkms")
			out = append(out, entry(p, "aur", kind(isKmod, fw), snap.HasAUR(p.ID)))
		}
	}
	return out
}

// depBase strips a dependency version constraint ("linux-headers>=6.0" ->
// "linux-headers").
func depBase(dep string) string {
	for _, sep := range []string{">=", "<=", "=", ">", "<"} {
		if i := strings.Index(dep, sep); i >= 0 {
			return dep[:i]
		}
	}
	return dep
}

// entry builds the AppEntry rendered by AppCard. Status reflects the installed
// flag so the card shows Installed vs available.
func entry(p models.SourcePackage, srcType, driverKind string, isInstalled bool) models.AppEntry {
	status := models.StatusAvailable
	if isInstalled {
		status = models.StatusInstalled
	}
	return models.AppEntry{
		ID:          strings.ToLower(p.ID),
		Name:        p.DisplayName,
		Publisher:   p.Publisher,
		Description: p.Description,
		Status:      status,
		DriverKind:  driverKind,
		Sources: []models.AppSource{{
			ID:            srcType + ":" + p.ID,
			Type:          srcType,
			PackageName:   p.ID,
			Version:       p.Version,
			SizeBytes:     p.SizeBytes,
			IsRecommended: true,
			Installed:     isInstalled,
		}},
	}
}
