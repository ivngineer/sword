package drivers

import (
	"context"
	"log"
	"os"
	"strings"

	"sword/backend/models"
)

// FirmwarePackage is one missing firmware package recommended for the
// machine's hardware. Name is the exact pacman package name, ready to pass to
// the installer verbatim.
type FirmwarePackage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	SizeBytes   int64  `json:"sizeBytes"`
}

// Firmware detects firmware packages recommended for the current hardware
// that are not yet installed. Probes are cheap file reads under /proc and
// /sys; the only subprocess work is two batched expac calls (local DB filter,
// sync-repo validation + metadata). Returns nil when nothing is missing.
func (d *Detector) Firmware(ctx context.Context) []FirmwarePackage {
	cands := firmwareCandidates()
	if len(cands) == 0 {
		return nil
	}

	// Drop anything already in the local DB (covers native and foreign).
	installedPkgs, err := d.pac.LocalQuery(ctx, cands)
	if err != nil {
		log.Printf("firmware: local query: %v", err)
	}
	have := make(map[string]bool, len(installedPkgs))
	for _, p := range installedPkgs {
		have[p.ID] = true
	}
	missing := make([]string, 0, len(cands))
	for _, n := range cands {
		if !have[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	// Keep only names that actually exist in the sync repos, picking up
	// description/version/size for the review dialog along the way.
	info, err := d.pac.SyncInfo(ctx, missing)
	if err != nil {
		log.Printf("firmware: sync info: %v", err)
		return nil
	}
	byName := make(map[string]models.SourcePackage, len(info))
	for _, p := range info {
		byName[p.ID] = p
	}
	out := make([]FirmwarePackage, 0, len(missing))
	for _, n := range missing {
		p, ok := byName[n]
		if !ok {
			continue
		}
		out = append(out, FirmwarePackage{
			Name:        p.ID,
			Description: p.Description,
			Version:     p.Version,
			SizeBytes:   p.SizeBytes,
		})
	}
	return out
}

// firmwareCandidates maps hardware probes to pacman package names. Every
// probe is a cheap /proc or /sys read; unknown hardware contributes nothing.
func firmwareCandidates() []string {
	// Device firmware blobs are relevant on effectively any machine.
	out := []string{"linux-firmware"}

	switch cpuVendor() {
	case "GenuineIntel":
		out = append(out, "intel-ucode")
	case "AuthenticAMD":
		out = append(out, "amd-ucode")
	}

	// Sound Open Firmware: the snd_sof driver stack is loaded but needs its
	// firmware payload package to bring the audio hardware up.
	if moduleLoaded("snd_sof") {
		out = append(out, "sof-firmware")
	}

	// Wireless regulatory database for any wifi adapter.
	if entries, err := os.ReadDir("/sys/class/ieee80211"); err == nil && len(entries) > 0 {
		out = append(out, "wireless-regdb")
	}
	return out
}

// cpuVendor returns the vendor_id from /proc/cpuinfo, e.g. "GenuineIntel".
func cpuVendor() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "vendor_id") {
			if i := strings.IndexByte(line, ':'); i >= 0 {
				return strings.TrimSpace(line[i+1:])
			}
		}
	}
	return ""
}

// moduleLoaded reports whether any loaded kernel module's name starts with
// prefix (first field per /proc/modules line).
func moduleLoaded(prefix string) bool {
	b, err := os.ReadFile("/proc/modules")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		name, _, _ := strings.Cut(line, " ")
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
