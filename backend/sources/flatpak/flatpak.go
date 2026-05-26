// Package flatpak is the Source backed by configured flatpak remotes. It
// shells out to `flatpak remote-ls` and parses tab-separated output.
package flatpak

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"

	"sword/backend/models"
	"sword/backend/sources/proc"
)

const sourceName = "flatpak"

// Source implements sources.Source for flatpak.
type Source struct{}

// New returns a flatpak Source.
func New() *Source { return &Source{} }

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

// Install installs a flatpak app from flathub. System-level installs trigger a
// polkit prompt; --user installs do not. We prefer --user so the action
// completes without auth where possible.
func (s *Source) Install(ctx context.Context, id string) error {
	return proc.RunDetached(ctx, "flatpak", "install", "-y", "--user", "flathub", id)
}

// Remove uninstalls a flatpak app from the user installation.
func (s *Source) Remove(ctx context.Context, id string) error {
	return proc.RunDetached(ctx, "flatpak", "uninstall", "-y", "--user", id)
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
