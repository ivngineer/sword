// Package metadata parses AppStream catalogs and resolves application icons.
package metadata

import (
	"compress/gzip"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AppStreamRecord is the subset of AppStream metadata Sword needs.
type AppStreamRecord struct {
	ID          string
	Name        string
	Summary     string
	Developer   string
	License     string
	PkgName     string
	IconURL     string
	Screenshots []string
}

// AppStreamResolver looks up a record by an AppStream component id (or, as a
// convenience, by a distro package name — implementations index both).
type AppStreamResolver interface {
	Lookup(key string) *AppStreamRecord
}

// flathubFeedURL is the canonical Flathub AppStream catalog.
const flathubFeedURL = "https://flathub.org/repo/appstream/x86_64/appstream.xml.gz"

// flathubIconBase is where Flathub serves cached 128px component icons.
const flathubIconBase = "https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/"

// --- XML model -------------------------------------------------------------

type xmlLang struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

type xmlIcon struct {
	Type  string `xml:"type,attr"`
	Width string `xml:"width,attr"`
	Value string `xml:",chardata"`
}

type xmlImage struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type xmlScreenshot struct {
	Type   string     `xml:"type,attr"`
	Images []xmlImage `xml:"image"`
}

type xmlComponent struct {
	Type           string    `xml:"type,attr"`
	ID             string    `xml:"id"`
	Names          []xmlLang `xml:"name"`
	Summaries      []xmlLang `xml:"summary"`
	PkgName        string    `xml:"pkgname"`
	DeveloperName  string    `xml:"developer_name"`
	Developer      struct {
		Name string `xml:"name"`
	} `xml:"developer"`
	ProjectLicense string          `xml:"project_license"`
	Icons          []xmlIcon       `xml:"icon"`
	Screenshots    []xmlScreenshot `xml:"screenshots>screenshot"`
}

type xmlComponents struct {
	Components []xmlComponent `xml:"component"`
}

func defaultLang(items []xmlLang) string {
	for _, it := range items {
		if it.Lang == "" || it.Lang == "C" || it.Lang == "en" || it.Lang == "en-US" {
			return strings.TrimSpace(it.Value)
		}
	}
	if len(items) > 0 {
		return strings.TrimSpace(items[0].Value)
	}
	return ""
}

func recordFromComponent(c xmlComponent) *AppStreamRecord {
	dev := strings.TrimSpace(c.DeveloperName)
	if dev == "" {
		dev = strings.TrimSpace(c.Developer.Name)
	}
	return &AppStreamRecord{
		ID:          strings.TrimSpace(c.ID),
		Name:        defaultLang(c.Names),
		Summary:     defaultLang(c.Summaries),
		Developer:   dev,
		License:     strings.TrimSpace(c.ProjectLicense),
		PkgName:     strings.TrimSpace(c.PkgName),
		Screenshots: screenshotsFromComponent(c),
	}
}

// screenshotsFromComponent extracts screenshot URLs in catalog order. Prefer
// the "source" (full-res) image; fall back to "thumbnail" when none is given.
// Default screenshot (Type="default") sorts to the front so the first image
// shown matches what app stores feature.
func screenshotsFromComponent(c xmlComponent) []string {
	if len(c.Screenshots) == 0 {
		return nil
	}
	pick := func(s xmlScreenshot) string {
		var thumb string
		for _, img := range s.Images {
			v := strings.TrimSpace(img.Value)
			if v == "" {
				continue
			}
			if img.Type == "source" {
				return v
			}
			if thumb == "" && img.Type == "thumbnail" {
				thumb = v
			}
		}
		return thumb
	}
	out := make([]string, 0, len(c.Screenshots))
	var rest []string
	for _, s := range c.Screenshots {
		u := pick(s)
		if u == "" {
			continue
		}
		if s.Type == "default" {
			out = append(out, u)
		} else {
			rest = append(rest, u)
		}
	}
	return append(out, rest...)
}

func indexRecord(m map[string]*AppStreamRecord, rec *AppStreamRecord) {
	if rec.ID != "" {
		id := strings.ToLower(rec.ID)
		m[id] = rec
		// Catalogs sometimes suffix desktop components with ".desktop";
		// flatpak/source ids never do. Index the bare form too.
		if bare := strings.TrimSuffix(id, ".desktop"); bare != id {
			if _, ok := m[bare]; !ok {
				m[bare] = rec
			}
		}
	}
	if rec.PkgName != "" {
		if _, exists := m[strings.ToLower(rec.PkgName)]; !exists {
			m[strings.ToLower(rec.PkgName)] = rec
		}
	}
}

func readMaybeGz(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r = gz
	}
	return io.ReadAll(r)
}

// --- LocalResolver ---------------------------------------------------------

// LocalResolver scans on-disk metainfo directories and resolves icons from
// the local hicolor icon theme.
type LocalResolver struct {
	mu      sync.RWMutex
	records map[string]*AppStreamRecord
}

// NewLocalResolver returns an empty LocalResolver; call Load to populate it.
func NewLocalResolver() *LocalResolver {
	return &LocalResolver{records: map[string]*AppStreamRecord{}}
}

// Lookup returns the record for an AppStream id or package name, or nil.
func (r *LocalResolver) Lookup(key string) *AppStreamRecord {
	if key == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.records[strings.ToLower(key)]
}

// Load scans /usr/share/metainfo and /usr/share/appdata. It is a fast disk
// scan and is safe to call synchronously at startup.
func (r *LocalResolver) Load() {
	recs := map[string]*AppStreamRecord{}
	for _, dir := range []string{"/usr/share/metainfo", "/usr/share/appdata"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || (!strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".xml.gz")) {
				continue
			}
			path := filepath.Join(dir, name)
			data, err := readMaybeGz(path)
			if err != nil {
				log.Printf("appstream: read %s: %v", path, err)
				continue
			}
			var c xmlComponent
			if err := xml.Unmarshal(data, &c); err != nil || c.ID == "" {
				continue
			}
			rec := recordFromComponent(c)
			rec.IconURL = localIcon(c)
			indexRecord(recs, rec)
		}
	}
	r.mu.Lock()
	r.records = recs
	r.mu.Unlock()
	log.Printf("appstream: local resolver loaded %d records", len(recs))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// localIcon searches the hicolor theme for an on-disk icon and returns it as
// a file:// URL, or "" when nothing is found.
func localIcon(c xmlComponent) string {
	var names []string
	for _, ic := range c.Icons {
		v := strings.TrimSpace(ic.Value)
		if ic.Type == "local" && strings.HasPrefix(v, "/") {
			return "file://" + v
		}
		if v != "" {
			names = append(names, v)
		}
	}
	names = append(names, c.ID, c.PkgName)
	sizes := []string{"128x128", "256x256", "192x192", "96x96", "64x64", "48x48", "scalable"}
	for _, n := range names {
		if n == "" {
			continue
		}
		base := strings.TrimSuffix(n, filepath.Ext(n))
		for _, sz := range sizes {
			for _, ext := range []string{"png", "svg"} {
				p := filepath.Join("/usr/share/icons/hicolor", sz, "apps", base+"."+ext)
				if fileExists(p) {
					return "file://" + p
				}
			}
		}
		if p := filepath.Join("/usr/share/pixmaps", base+".png"); fileExists(p) {
			return "file://" + p
		}
	}
	return ""
}

// --- feed resolvers (remote catalogs) --------------------------------------

// feedResolver fetches a remote AppStream collection catalog once at startup.
type feedResolver struct {
	name     string
	url      string
	iconFunc func(xmlComponent) string

	mu      sync.RWMutex
	records map[string]*AppStreamRecord
}

// Lookup returns the record for an AppStream id or package name, or nil.
func (r *feedResolver) Lookup(key string) *AppStreamRecord {
	if key == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.records[strings.ToLower(key)]
}

// Load fetches and parses the remote catalog. A failed fetch degrades the
// resolver to a no-op rather than crashing the backend.
func (r *feedResolver) Load() {
	if r.url == "" {
		log.Printf("appstream: %s feed disabled (no URL configured)", r.name)
		return
	}
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(r.url)
	if err != nil {
		log.Printf("appstream: %s feed fetch: %v", r.name, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("appstream: %s feed status %s", r.name, resp.Status)
		return
	}
	var reader io.Reader = resp.Body
	if strings.HasSuffix(r.url, ".gz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("appstream: %s feed gunzip: %v", r.name, err)
			return
		}
		defer gz.Close()
		reader = gz
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("appstream: %s feed read: %v", r.name, err)
		return
	}
	var cs xmlComponents
	if err := xml.Unmarshal(data, &cs); err != nil {
		log.Printf("appstream: %s feed parse: %v", r.name, err)
		return
	}
	recs := map[string]*AppStreamRecord{}
	for _, c := range cs.Components {
		if c.ID == "" {
			continue
		}
		rec := recordFromComponent(c)
		rec.IconURL = r.iconFunc(c)
		indexRecord(recs, rec)
	}
	r.mu.Lock()
	r.records = recs
	r.mu.Unlock()
	log.Printf("appstream: %s feed loaded %d records", r.name, len(recs))
}

// NewDistroFeedResolver returns the Arch Linux AppStream feed resolver. Arch
// has no single canonical web feed, so the URL is taken from the
// SWORD_DISTRO_FEED_URL environment variable; when unset the resolver is a
// graceful no-op.
func NewDistroFeedResolver() *feedResolver {
	return &feedResolver{
		name:     "distro",
		url:      os.Getenv("SWORD_DISTRO_FEED_URL"),
		iconFunc: distroIcon,
		records:  map[string]*AppStreamRecord{},
	}
}

// NewFlathubFeedResolver returns the Flathub AppStream feed resolver.
func NewFlathubFeedResolver() *feedResolver {
	return &feedResolver{
		name:     "flathub",
		url:      flathubFeedURL,
		iconFunc: flathubIcon,
		records:  map[string]*AppStreamRecord{},
	}
}

func flathubIcon(c xmlComponent) string {
	var fallback string
	for _, ic := range c.Icons {
		v := strings.TrimSpace(ic.Value)
		switch ic.Type {
		case "cached":
			if ic.Width == "128" || ic.Width == "" {
				return flathubIconBase + v
			}
			fallback = flathubIconBase + v
		case "remote":
			if strings.HasPrefix(v, "http") {
				fallback = v
			}
		}
	}
	return fallback
}

func distroIcon(c xmlComponent) string {
	for _, ic := range c.Icons {
		v := strings.TrimSpace(ic.Value)
		if ic.Type == "remote" && strings.HasPrefix(v, "http") {
			return v
		}
	}
	return ""
}

// Resolve returns the first record found for any of keys, searching resolvers
// in order.
func Resolve(resolvers []AppStreamResolver, keys ...string) *AppStreamRecord {
	for _, k := range keys {
		if k == "" {
			continue
		}
		for _, r := range resolvers {
			if rec := r.Lookup(k); rec != nil {
				return rec
			}
		}
	}
	return nil
}
