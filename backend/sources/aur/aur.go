// Package aur is the Source backed by the AUR RPC v5 HTTP API.
package aur

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"sword/backend/models"
	"sword/backend/sources"
	"sword/backend/sources/proc"
)

var _ sources.Source = (*Source)(nil)

const (
	sourceName = "aur"
	rpcBase    = "https://aur.archlinux.org/rpc/v5"
)

// Source implements sources.Source for the AUR.
type Source struct {
	client *http.Client
}

// New returns an AUR Source.
func New() *Source {
	return &Source{client: &http.Client{Timeout: 10 * time.Second}}
}

// Name returns "aur".
func (s *Source) Name() string { return sourceName }

type rpcResult struct {
	Name        string `json:"Name"`
	Version     string `json:"Version"`
	Description string `json:"Description"`
	Maintainer  string `json:"Maintainer"`
}

type rpcResponse struct {
	Type    string      `json:"type"`
	Error   string      `json:"error"`
	Results []rpcResult `json:"results"`
}

// Search queries the AUR by name and description. The AUR rejects queries
// shorter than two characters, so those return no results.
func (s *Source) Search(ctx context.Context, query string) ([]models.SourcePackage, error) {
	if len(query) < 2 {
		return nil, nil
	}
	u := rpcBase + "/search/" + url.PathEscape(query) + "?by=name-desc"
	resp, err := s.fetch(ctx, u)
	if err != nil {
		return nil, err
	}
	return toPackages(resp.Results), nil
}

// Get returns a single AUR package by name.
func (s *Source) Get(ctx context.Context, id string) (models.SourcePackage, error) {
	u := rpcBase + "/info?arg[]=" + url.QueryEscape(id)
	resp, err := s.fetch(ctx, u)
	if err != nil {
		return models.SourcePackage{}, err
	}
	pkgs := toPackages(resp.Results)
	if len(pkgs) == 0 {
		return models.SourcePackage{}, errors.New("aur: package not found: " + id)
	}
	return pkgs[0], nil
}

// Install builds and installs an AUR package using paru or yay. Detached so
// the helper's pacman invocation routes its sudo/polkit prompt through the
// session auth agent rather than /dev/tty.
func (s *Source) Install(ctx context.Context, id string, onProgress sources.ProgressFn) error {
	helper, err := findHelper()
	if err != nil {
		return err
	}
	return proc.RunStreaming(ctx, lineFn(onProgress), helper, "-S", "--noconfirm", id)
}

// Remove uninstalls an AUR-installed package using paru or yay.
func (s *Source) Remove(ctx context.Context, id string, onProgress sources.ProgressFn) error {
	helper, err := findHelper()
	if err != nil {
		return err
	}
	return proc.RunStreaming(ctx, lineFn(onProgress), helper, "-Rs", "--noconfirm", id)
}

func lineFn(onProgress sources.ProgressFn) func(string) {
	return func(line string) {
		frac, status := proc.ParseProgress(line)
		onProgress(frac, status)
	}
}

func findHelper() (string, error) {
	for _, h := range []string{"paru", "yay"} {
		if p, err := exec.LookPath(h); err == nil {
			return p, nil
		}
	}
	return "", errors.New("aur: no AUR helper found (install paru or yay)")
}

func (s *Source) fetch(ctx context.Context, u string) (*rpcResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("aur: rpc status " + res.Status)
	}
	var parsed rpcResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if parsed.Type == "error" {
		return nil, errors.New("aur: " + parsed.Error)
	}
	return &parsed, nil
}

func toPackages(results []rpcResult) []models.SourcePackage {
	pkgs := make([]models.SourcePackage, 0, len(results))
	for _, r := range results {
		pkgs = append(pkgs, models.SourcePackage{
			SourceName:  sourceName,
			ID:          r.Name,
			DisplayName: r.Name,
			Publisher:   r.Maintainer,
			Version:     r.Version,
			Description: r.Description,
		})
	}
	return pkgs
}
