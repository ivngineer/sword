// Package pacman is the Source backed by the Arch sync databases. It shells
// out to `expac` and parses tab-separated output.
package pacman

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"sword/backend/models"
	"sword/backend/sources"
	"sword/backend/sources/proc"
)

var _ sources.Source = (*Source)(nil)

const sourceName = "pacman"

// Source implements sources.Source for pacman/expac.
type Source struct{}

// New returns a pacman Source.
func New() *Source { return &Source{} }

// Name returns "pacman".
func (s *Source) Name() string { return sourceName }

// Available reports whether the expac binary is on PATH.
func (s *Source) Available() bool {
	_, err := exec.LookPath("expac")
	return err == nil
}

// Search runs `expac -Ss <query>`; an empty query matches every package.
func (s *Source) Search(ctx context.Context, query string) ([]models.SourcePackage, error) {
	if !s.Available() {
		return nil, errors.New("pacman: expac not installed")
	}
	// %n name, %v version, %d description, %m installed size in bytes.
	cmd := exec.CommandContext(ctx, "expac", "-Ss", `%n\t%v\t%d\t%m`, query)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		// expac exits non-zero when nothing matched: treat as empty.
		if out.Len() == 0 {
			return nil, nil
		}
	}
	return parse(out.Bytes()), nil
}

// Get returns a single package by name.
func (s *Source) Get(ctx context.Context, id string) (models.SourcePackage, error) {
	if !s.Available() {
		return models.SourcePackage{}, errors.New("pacman: expac not installed")
	}
	cmd := exec.CommandContext(ctx, "expac", "-S", `%n\t%v\t%d\t%m`, id)
	out, err := cmd.Output()
	if err != nil {
		return models.SourcePackage{}, err
	}
	pkgs := parse(out)
	if len(pkgs) == 0 {
		return models.SourcePackage{}, errors.New("pacman: package not found: " + id)
	}
	return pkgs[0], nil
}

// LocalQuery returns metadata for already-installed packages by name, reading
// the local pacman DB (`expac -Q`). Useful for surfacing AUR-installed
// packages that do not appear in any sync repo. Unknown names are silently
// dropped. Empty input returns nil.
func (s *Source) LocalQuery(ctx context.Context, names []string) ([]models.SourcePackage, error) {
	if len(names) == 0 {
		return nil, nil
	}
	if !s.Available() {
		return nil, errors.New("pacman: expac not installed")
	}
	args := append([]string{"-Q", `%n\t%v\t%d\t%m`}, names...)
	cmd := exec.CommandContext(ctx, "expac", args...)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, nil
	}
	return parse(out), nil
}

// SyncInfo returns sync-repo metadata for the named packages. Unknown names
// are silently dropped; empty input returns nil.
func (s *Source) SyncInfo(ctx context.Context, names []string) ([]models.SourcePackage, error) {
	if len(names) == 0 {
		return nil, nil
	}
	if !s.Available() {
		return nil, errors.New("pacman: expac not installed")
	}
	args := append([]string{"-S", `%n\t%v\t%d\t%m`}, names...)
	cmd := exec.CommandContext(ctx, "expac", args...)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, nil
	}
	return parse(out), nil
}

// Deps returns the dependency list for each named sync package, keyed by
// package name. Runs `expac -S '%n\t%E' names...` (%E = depends-on). Unknown
// names are silently dropped; empty input returns an empty map.
func (s *Source) Deps(ctx context.Context, names []string) (map[string][]string, error) {
	if len(names) == 0 {
		return map[string][]string{}, nil
	}
	if !s.Available() {
		return nil, errors.New("pacman: expac not installed")
	}
	args := append([]string{"-S", `%n\t%E`}, names...)
	cmd := exec.CommandContext(ctx, "expac", args...)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return map[string][]string{}, nil
	}
	deps := make(map[string][]string, len(names))
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		f := strings.SplitN(line, "\t", 2)
		if len(f) < 1 || f[0] == "" {
			continue
		}
		var list []string
		if len(f) == 2 {
			list = strings.Fields(f[1])
		}
		deps[f[0]] = list
	}
	return deps, nil
}

// Install installs a package via pkexec + pacman. Runs detached so pkexec
// routes auth through the session polkit agent instead of /dev/tty.
func (s *Source) Install(ctx context.Context, id string, onProgress sources.ProgressFn) error {
	return proc.RunStreaming(ctx, lineFn(onProgress),
		"pkexec", "pacman", "-S", "--noconfirm", id)
}

// InstallMany installs several packages in one pacman transaction: a single
// pkexec auth prompt and shared dependency resolution. --needed skips any
// package that is already up to date.
func (s *Source) InstallMany(ctx context.Context, ids []string, onProgress sources.ProgressFn) error {
	args := append([]string{"pacman", "-S", "--noconfirm", "--needed"}, ids...)
	return proc.RunStreaming(ctx, lineFn(onProgress), "pkexec", args...)
}

// Remove uninstalls a package via pkexec + pacman.
func (s *Source) Remove(ctx context.Context, id string, onProgress sources.ProgressFn) error {
	return proc.RunStreaming(ctx, lineFn(onProgress),
		"pkexec", "pacman", "-Rs", "--noconfirm", id)
}

func lineFn(onProgress sources.ProgressFn) func(string) {
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
		var size int64
		if len(f) >= 4 {
			size, _ = strconv.ParseInt(strings.TrimSpace(f[3]), 10, 64)
		}
		pkgs = append(pkgs, models.SourcePackage{
			SourceName:  sourceName,
			ID:          f[0],
			DisplayName: f[0],
			Version:     f[1],
			Description: f[2],
			SizeBytes:   size,
		})
	}
	return pkgs
}
