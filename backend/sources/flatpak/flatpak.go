// Package flatpak is the Source backed by configured flatpak remotes. It
// shells out to `flatpak remote-ls` and parses tab-separated output.
package flatpak

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log"
	"os/exec"
	"strings"
	"sync"

	"sword/backend/models"
	"sword/backend/sources"
	"sword/backend/sources/proc"
)

var _ sources.Source = (*Source)(nil)

const sourceName = "flatpak"
const flathubRepoURL = "https://dl.flathub.org/repo/flathub.flatpakrepo"

// Source implements sources.Source for flatpak.
type Source struct {
	remoteOnce sync.Once
}

// New returns a flatpak Source and kicks off a one-shot bootstrap that adds
// the flathub remote at user scope if it isn't already configured. Without a
// user-scoped flathub remote, `flatpak install --user flathub <id>` fails
// with "No remote refs found for 'flathub'" even when flathub exists at
// system scope.
func New() *Source {
	s := &Source{}
	go s.ensureFlathubRemote(context.Background())
	return s
}

// ensureFlathubRemote runs `flatpak remote-add --user --if-not-exists` once
// per process. The flag makes the call an idempotent no-op when the remote is
// already present (no network round-trip).
func (s *Source) ensureFlathubRemote(ctx context.Context) {
	s.remoteOnce.Do(func() {
		if !s.Available() {
			return
		}
		cmd := exec.CommandContext(ctx, "flatpak", "remote-add",
			"--user", "--if-not-exists", "flathub", flathubRepoURL)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			log.Printf("flatpak: remote-add flathub: %v: %s", err, strings.TrimSpace(stderr.String()))
		}
	})
}

// Name returns "flatpak".
func (s *Source) Name() string { return sourceName }

// Available reports whether the flatpak binary is on PATH.
func (s *Source) Available() bool {
	_, err := exec.LookPath("flatpak")
	return err == nil
}

// Search lists every remote application and substring-filters by query.
// remote-ls has no server-side search, so filtering happens locally.
func (s *Source) Search(ctx context.Context, query string) ([]models.SourcePackage, error) {
	if !s.Available() {
		return nil, errors.New("flatpak: binary not installed")
	}
	cmd := exec.CommandContext(ctx, "flatpak", "remote-ls",
		"--columns=application,name,version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	all := parse(out.Bytes())
	if query == "" {
		return all, nil
	}
	q := strings.ToLower(query)
	var matched []models.SourcePackage
	for _, p := range all {
		if strings.Contains(strings.ToLower(p.DisplayName), q) ||
			strings.Contains(strings.ToLower(p.AppStreamID), q) {
			matched = append(matched, p)
		}
	}
	return matched, nil
}

// Get returns a single flatpak app by its application id.
func (s *Source) Get(ctx context.Context, id string) (models.SourcePackage, error) {
	all, err := s.Search(ctx, "")
	if err != nil {
		return models.SourcePackage{}, err
	}
	for _, p := range all {
		if p.AppStreamID == id {
			return p, nil
		}
	}
	return models.SourcePackage{}, errors.New("flatpak: package not found: " + id)
}

// Install installs a flatpak app from flathub at user scope. System-level
// installs trigger a polkit prompt; --user installs do not. --or-update makes
// re-install of an already-present app a no-op upgrade instead of an error.
// --noninteractive forces deterministic exit codes off a tty.
func (s *Source) Install(ctx context.Context, id string, onProgress sources.ProgressFn) error {
	s.ensureFlathubRemote(ctx)
	return proc.RunStreaming(ctx, progressLine(onProgress), "flatpak", "install",
		"-y", "--noninteractive", "--user", "--or-update", "flathub", id)
}

// Remove uninstalls a flatpak app. Tries user scope first (no auth prompt);
// if the app isn't installed at user scope, falls back to system scope which
// routes a polkit prompt through the session auth agent.
func (s *Source) Remove(ctx context.Context, id string, onProgress sources.ProgressFn) error {
	onLine := progressLine(onProgress)
	if err := proc.RunStreaming(ctx, onLine, "flatpak", "uninstall",
		"-y", "--noninteractive", "--user", id); err == nil {
		return nil
	}
	return proc.RunStreaming(ctx, onLine, "flatpak", "uninstall",
		"-y", "--noninteractive", "--system", id)
}

func progressLine(onProgress sources.ProgressFn) func(string) {
	return func(line string) {
		frac, status := proc.ParseProgress(line)
		onProgress(frac, status)
	}
}

func parse(b []byte) []models.SourcePackage {
	var pkgs []models.SourcePackage
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 3 {
			continue
		}
		appID, name, version := f[0], f[1], f[2]
		if name == "" {
			name = appID
		}
		pkgs = append(pkgs, models.SourcePackage{
			SourceName:  sourceName,
			ID:          appID,
			DisplayName: name,
			Version:     version,
			// A flatpak application id is also its AppStream component id.
			AppStreamID: appID,
		})
	}
	return pkgs
}
