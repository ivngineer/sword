// Package updates checks for available package updates across pacman, the
// AUR and flatpak, and runs the combined full-system update. Like
// drivers.Detector it is cross-source orchestration that lives beside the
// sources.Source interface rather than inside it.
package updates

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"sword/backend/sources"
	"sword/backend/sources/aur"
	"sword/backend/sources/proc"
)

// Entry is one available update, shaped for direct JSON serialization.
type Entry struct {
	// Name is the display name; defaults to PackageName until the IPC layer
	// stamps a friendlier one from the registry.
	Name string `json:"name"`
	// PackageName is the pacman package name or flatpak application id —
	// the identifier passed to the underlying tool for update/skip.
	PackageName    string `json:"packageName"`
	SourceType     string `json:"sourceType"` // "pacman" | "aur" | "flatpak"
	IconURL        string `json:"iconUrl"`    // "" when the app has no .desktop entry / icon
	CurrentVersion string `json:"currentVersion"`
	NewVersion     string `json:"newVersion"`
}

// Skip carries per-manager ignore lists for a full system update.
type Skip struct {
	Pacman  []string `json:"pacman"`
	AUR     []string `json:"aur"`
	Flatpak []string `json:"flatpak"`
}

// Checker queries the three managers for pending updates and runs the
// combined upgrade. The most recent Check result is cached because flatpak
// has no --ignore flag: skips translate into an explicit target list drawn
// from that cache.
type Checker struct {
	client *http.Client

	lastMu sync.Mutex
	last   []Entry
}

// NewChecker returns a Checker.
func NewChecker() *Checker {
	return &Checker{client: &http.Client{Timeout: 10 * time.Second}}
}

// Check queries all three managers concurrently and returns the merged,
// name-sorted list. Each manager degrades to an empty slice on error
// (missing checkupdates, no network, no flatpak) — Check never fails.
func (c *Checker) Check(ctx context.Context) []Entry {
	var pac, au, fp []Entry
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); pac = pacmanUpdates(ctx) }()
	go func() { defer wg.Done(); au = c.aurUpdates(ctx) }()
	go func() { defer wg.Done(); fp = flatpakUpdates(ctx) }()
	wg.Wait()

	all := make([]Entry, 0, len(pac)+len(au)+len(fp))
	all = append(all, pac...)
	all = append(all, au...)
	all = append(all, fp...)
	sort.Slice(all, func(i, j int) bool {
		return strings.ToLower(all[i].Name) < strings.ToLower(all[j].Name)
	})

	c.lastMu.Lock()
	c.last = all
	c.lastMu.Unlock()
	return all
}

// cached returns the most recent Check result.
func (c *Checker) cached() []Entry {
	c.lastMu.Lock()
	defer c.lastMu.Unlock()
	return c.last
}

// --- per-manager checks ------------------------------------------------------

// pacmanUpdates runs `checkupdates` (pacman-contrib) when available and
// otherwise falls back to replicating it: sync the repo databases into a
// throwaway copy and diff against the installed versions, never touching
// /var/lib/pacman. Exit code 2 from checkupdates means "no updates".
func pacmanUpdates(ctx context.Context) []Entry {
	if _, err := exec.LookPath("checkupdates"); err != nil {
		return pacmanUpdatesFallback(ctx)
	}
	cmd := exec.CommandContext(ctx, "checkupdates", "--nocolor")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 2 {
			return nil // no updates pending
		}
		log.Printf("updates: checkupdates: %v: %s", err, strings.TrimSpace(stderr.String()))
		return nil
	}
	return parsePacmanUpdateLines(&out)
}

// pacmanUpdatesFallback is what checkupdates does under the hood: copy of the
// sync databases in a temp dbpath (the local db is symlinked in), refreshed
// with `fakeroot pacman -Sy`, then `pacman -Qu` against it. Requires fakeroot
// because pacman refuses -Sy as non-root even with a writable dbpath.
func pacmanUpdatesFallback(ctx context.Context) []Entry {
	if _, err := exec.LookPath("fakeroot"); err != nil {
		log.Printf("updates: neither checkupdates nor fakeroot installed; pacman updates unavailable")
		return nil
	}
	db := filepath.Join(os.TempDir(), fmt.Sprintf("sword-checkup-db-%d", os.Getuid()))
	if err := os.MkdirAll(db, 0o755); err != nil {
		log.Printf("updates: pacman fallback: %v", err)
		return nil
	}
	// -Qu needs the real local db to know what's installed.
	if err := os.Symlink("/var/lib/pacman/local", filepath.Join(db, "local")); err != nil && !os.IsExist(err) {
		log.Printf("updates: pacman fallback: %v", err)
		return nil
	}
	// --disable-sandbox: pacman 7's download sandbox (Landlock/priv-drop)
	// cannot apply under fakeroot; upstream checkupdates does the same.
	sync := exec.CommandContext(ctx, "fakeroot", "--", "pacman", "-Sy",
		"--dbpath", db, "--logfile", "/dev/null", "--disable-sandbox")
	var stderr bytes.Buffer
	sync.Stderr = &stderr
	if err := sync.Run(); err != nil {
		log.Printf("updates: pacman fallback -Sy: %v: %s", err, strings.TrimSpace(stderr.String()))
		return nil
	}
	out, err := exec.CommandContext(ctx, "pacman", "-Qu", "--dbpath", db, "--color", "never").Output()
	if err != nil {
		// -Qu exits 1 when there is nothing to upgrade.
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 1 {
			log.Printf("updates: pacman fallback -Qu: %v", err)
		}
		return nil
	}
	return parsePacmanUpdateLines(bytes.NewReader(out))
}

// parsePacmanUpdateLines parses "name oldver -> newver" lines as emitted by
// both checkupdates and pacman -Qu. Lines flagged [ignored] are dropped.
func parsePacmanUpdateLines(r io.Reader) []Entry {
	var entries []Entry
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if strings.Contains(sc.Text(), "[ignored]") {
			continue
		}
		f := strings.Fields(sc.Text())
		if len(f) != 4 || f[2] != "->" {
			continue
		}
		entries = append(entries, Entry{
			Name:           f[0],
			PackageName:    f[0],
			SourceType:     "pacman",
			CurrentVersion: f[1],
			NewVersion:     f[3],
		})
	}
	return entries
}

// aurUpdates lists foreign packages (`pacman -Qm`), fetches their AUR
// metadata via the RPC v5 info endpoint and keeps the ones whose AUR version
// is newer than the local one.
func (c *Checker) aurUpdates(ctx context.Context) []Entry {
	if _, err := exec.LookPath("pacman"); err != nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "pacman", "-Qm").Output()
	if err != nil {
		return nil
	}
	local := map[string]string{} // name -> installed version
	var names []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) != 2 {
			continue
		}
		local[f[0]] = f[1]
		names = append(names, f[0])
	}
	if len(names) == 0 {
		return nil
	}

	remote := c.aurInfo(ctx, names)
	var entries []Entry
	for name, cur := range local {
		next, ok := remote[name]
		if !ok || next == cur { // dropped from AUR, or up to date
			continue
		}
		if !versionNewer(ctx, cur, next) {
			continue
		}
		entries = append(entries, Entry{
			Name:           name,
			PackageName:    name,
			SourceType:     "aur",
			CurrentVersion: cur,
			NewVersion:     next,
		})
	}
	return entries
}

const aurRPCInfo = "https://aur.archlinux.org/rpc/v5/info"

// aurInfo fetches name->version for the given packages from the AUR RPC,
// chunked to keep URLs short. Failures degrade to whatever was fetched.
func (c *Checker) aurInfo(ctx context.Context, names []string) map[string]string {
	versions := map[string]string{}
	const chunk = 150
	for start := 0; start < len(names); start += chunk {
		end := min(start+chunk, len(names))
		q := url.Values{}
		for _, n := range names[start:end] {
			q.Add("arg[]", n)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, aurRPCInfo+"?"+q.Encode(), nil)
		if err != nil {
			continue
		}
		res, err := c.client.Do(req)
		if err != nil {
			log.Printf("updates: aur rpc: %v", err)
			continue
		}
		var parsed struct {
			Results []struct {
				Name    string `json:"Name"`
				Version string `json:"Version"`
			} `json:"results"`
		}
		err = json.NewDecoder(res.Body).Decode(&parsed)
		res.Body.Close()
		if err != nil {
			log.Printf("updates: aur rpc decode: %v", err)
			continue
		}
		for _, r := range parsed.Results {
			versions[r.Name] = r.Version
		}
	}
	return versions
}

// versionNewer reports whether next is newer than cur using pacman's vercmp.
// When vercmp is unavailable any difference counts as an update.
func versionNewer(ctx context.Context, cur, next string) bool {
	if _, err := exec.LookPath("vercmp"); err != nil {
		return cur != next
	}
	out, err := exec.CommandContext(ctx, "vercmp", cur, next).Output()
	if err != nil {
		return cur != next
	}
	return strings.TrimSpace(string(out)) == "-1"
}

// flatpakUpdates lists refs with pending updates across all installations.
// Runtimes are included (most flatpak updates are GL/Platform runtimes, and
// `flatpak update` upgrades them too). `remote-ls --updates` reports the new
// version; current versions come from `flatpak list`.
func flatpakUpdates(ctx context.Context) []Entry {
	if _, err := exec.LookPath("flatpak"); err != nil {
		return nil
	}
	current := map[string]string{} // ref id -> installed version
	if out, err := exec.CommandContext(ctx, "flatpak", "list",
		"--columns=application,version").Output(); err == nil {
		sc := bufio.NewScanner(bytes.NewReader(out))
		for sc.Scan() {
			f := strings.Split(sc.Text(), "\t")
			if len(f) >= 2 && f[0] != "" {
				current[f[0]] = f[1]
			}
		}
	}

	out, err := exec.CommandContext(ctx, "flatpak", "remote-ls", "--updates",
		"--columns=application,name,version").Output()
	if err != nil {
		log.Printf("updates: flatpak remote-ls --updates: %v", err)
		return nil
	}
	var entries []Entry
	seen := map[string]struct{}{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 2 || f[0] == "" {
			continue
		}
		appID, name := f[0], f[1]
		version := "" // runtimes often report no version
		if len(f) >= 3 {
			version = f[2]
		}
		if _, dup := seen[appID]; dup {
			continue
		}
		seen[appID] = struct{}{}
		if name == "" {
			name = appID
		}
		entries = append(entries, Entry{
			Name:           name,
			PackageName:    appID,
			SourceType:     "flatpak",
			CurrentVersion: current[appID],
			NewVersion:     version,
		})
	}
	return entries
}

// --- full system update ------------------------------------------------------

// RunFullUpdate upgrades everything not listed in skip. The Arch track
// (pacman + AUR, which share the pacman DB lock) and the flatpak track run in
// parallel; a failure in one does not stop the other and the errors are
// joined. onProgress receives the aggregated cross-track progress.
func (c *Checker) RunFullUpdate(ctx context.Context, skip Skip, onProgress sources.ProgressFn) error {
	last := c.cached()
	var hasPacman, hasAUR bool
	var flatpakIDs []string
	for _, e := range last {
		switch e.SourceType {
		case "pacman":
			hasPacman = true
		case "aur":
			hasAUR = true
		case "flatpak":
			flatpakIDs = append(flatpakIDs, e.PackageName)
		}
	}

	agg := newProgressAggregator(onProgress)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var errs []error
	runTrack := func(name string, run func(onLine func(string)) error) {
		track := agg.track(name)
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := run(track.line)
			track.finish()
			if err != nil {
				errMu.Lock()
				errs = append(errs, errors.New(name+": "+err.Error()))
				errMu.Unlock()
			}
		}()
	}

	if hasPacman || hasAUR {
		ignore := append(append([]string{}, skip.Pacman...), skip.AUR...)
		args := []string{"-Syu", "--noconfirm"}
		if len(ignore) > 0 {
			args = append(args, "--ignore", strings.Join(ignore, ","))
		}
		helper, herr := aur.FindHelper()
		runTrack("pacman", func(onLine func(string)) error {
			// One helper transaction covers repo + AUR; without a helper,
			// plain pacman upgrades the repos and AUR packages stay pending.
			if hasAUR && herr == nil {
				return proc.RunStreaming(ctx, onLine, helper, args...)
			}
			return proc.RunStreaming(ctx, onLine, "pkexec", append([]string{"pacman"}, args...)...)
		})
	}

	if len(flatpakIDs) > 0 {
		targets := flatpakTargets(flatpakIDs, skip.Flatpak)
		if len(skip.Flatpak) == 0 || len(targets) > 0 {
			runTrack("flatpak", func(onLine func(string)) error {
				args := []string{"update", "-y", "--noninteractive"}
				if len(skip.Flatpak) > 0 {
					// flatpak has no --ignore: skips become an explicit list.
					args = append(args, targets...)
				}
				return proc.RunStreaming(ctx, onLine, "flatpak", args...)
			})
		}
	}

	wg.Wait()
	return errors.Join(errs...)
}

// flatpakTargets returns ids minus the skipped ones.
func flatpakTargets(ids, skip []string) []string {
	skipped := make(map[string]struct{}, len(skip))
	for _, s := range skip {
		skipped[s] = struct{}{}
	}
	var out []string
	for _, id := range ids {
		if _, ok := skipped[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

// --- progress aggregation ----------------------------------------------------

// progressAggregator merges per-track progress into one stream. Overall
// fraction is (finished tracks + sum of active fractions) / track count;
// while any active track is indeterminate the overall is indeterminate too.
type progressAggregator struct {
	onProgress sources.ProgressFn

	mu     sync.Mutex
	tracks []*progressTrack
}

type progressTrack struct {
	agg      *progressAggregator
	name     string
	fraction float64 // -1 = indeterminate
	done     bool
}

func newProgressAggregator(onProgress sources.ProgressFn) *progressAggregator {
	return &progressAggregator{onProgress: onProgress}
}

func (a *progressAggregator) track(name string) *progressTrack {
	a.mu.Lock()
	defer a.mu.Unlock()
	t := &progressTrack{agg: a, name: name, fraction: -1}
	a.tracks = append(a.tracks, t)
	return t
}

func (t *progressTrack) line(line string) {
	frac, status := proc.ParseProgress(line)
	a := t.agg
	a.mu.Lock()
	if frac >= 0 {
		t.fraction = frac
	}
	overall := a.overallLocked()
	a.mu.Unlock()
	a.onProgress(overall, t.name+": "+status)
}

func (t *progressTrack) finish() {
	a := t.agg
	a.mu.Lock()
	t.done = true
	overall := a.overallLocked()
	a.mu.Unlock()
	a.onProgress(overall, t.name+": done")
}

func (a *progressAggregator) overallLocked() float64 {
	sum := 0.0
	for _, t := range a.tracks {
		switch {
		case t.done:
			sum += 1
		case t.fraction < 0:
			return -1
		default:
			sum += t.fraction
		}
	}
	return sum / float64(len(a.tracks))
}
