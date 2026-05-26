// Command sword-backend is the Sword search backend. It runs as a Tauri
// sidecar and speaks line-delimited JSON over stdin/stdout. Stdout carries
// only IPC messages; all logging goes to stderr.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"sword/backend/metadata"
	"sword/backend/models"
	"sword/backend/registry"
	"sword/backend/search"
	"sword/backend/sources"
	"sword/backend/sources/aur"
	"sword/backend/sources/flatpak"
	"sword/backend/sources/pacman"
	"sword/backend/sources/proc"
)

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	proc.EnsurePolkitAgent()

	pac := pacman.New()
	fp := flatpak.New()
	au := aur.New()

	for _, in := range []models.SourceInfo{
		{Name: "pacman", Available: pac.Available()},
		{Name: "flatpak", Available: fp.Available()},
		{Name: "aur", Available: true},
	} {
		log.Printf("source %s available=%v", in.Name, in.Available)
	}

	local := metadata.NewLocalResolver()
	distro := metadata.NewDistroFeedResolver()
	flathub := metadata.NewFlathubFeedResolver()
	resolvers := []metadata.AppStreamResolver{local, distro, flathub}

	// Local metainfo is a fast disk scan; remote feeds load concurrently so
	// the IPC server can start serving immediately.
	local.Load()
	var feeds sync.WaitGroup
	feeds.Add(2)
	go func() { defer feeds.Done(); distro.Load() }()
	go func() { defer feeds.Done(); flathub.Load() }()

	index := registry.NewAppIndex([]sources.Source{pac, fp}, resolvers)
	go func() {
		index.Build(context.Background())
		index.StartAutoRebuild(context.Background(), 30*time.Minute)
	}()
	// Re-build once the remote feeds are in so the index carries their
	// metadata and icons rather than waiting for the 30-minute cycle.
	go func() {
		feeds.Wait()
		index.Build(context.Background())
		log.Printf("registry: index re-built after feed load")
	}()

	orch := search.NewOrchestrator(index, au, resolvers)
	srcMap := map[string]sources.Source{
		pac.Name(): pac,
		fp.Name():  fp,
		au.Name():  au,
	}
	newServer(orch, index, srcMap).run()
}

// --- IPC protocol types ----------------------------------------------------

type inbound struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Query       string `json:"query"`
	AppID       string `json:"app_id"`
	SourceType  string `json:"source_type"`
	PackageName string `json:"package_name"`
}

type searchOut struct {
	Type    string            `json:"type"`
	ID      string            `json:"id"`
	Phase   string            `json:"phase"`
	Results []models.AppEntry `json:"results"`
}

type appDetailOut struct {
	Type string           `json:"type"`
	ID   string           `json:"id"`
	App  *models.AppEntry `json:"app"`
}

type popularOut struct {
	Type    string            `json:"type"`
	ID      string            `json:"id"`
	Results []models.AppEntry `json:"results"`
}

type errorOut struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

type actionOut struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	OK   bool   `json:"ok"`
}

// --- server ----------------------------------------------------------------

type server struct {
	orch  *search.Orchestrator
	index *registry.AppIndex
	srcs  map[string]sources.Source

	encMu sync.Mutex
	enc   *json.Encoder

	curMu  sync.Mutex
	cancel context.CancelFunc
}

func newServer(orch *search.Orchestrator, index *registry.AppIndex, srcs map[string]sources.Source) *server {
	return &server{orch: orch, index: index, srcs: srcs, enc: json.NewEncoder(os.Stdout)}
}

// send writes one JSON message followed by a newline. Encoder access is
// serialized so concurrent handlers cannot interleave output.
func (s *server) send(v any) {
	s.encMu.Lock()
	defer s.encMu.Unlock()
	if err := s.enc.Encode(v); err != nil {
		log.Printf("ipc: encode: %v", err)
	}
}

// run reads one JSON message per line from stdin and dispatches it. Each
// request is handled in its own goroutine so a slow request cannot block the
// reader.
func (s *server) run() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := append([]byte(nil), sc.Bytes()...)
		var msg inbound
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("ipc: bad message: %v", err)
			continue
		}
		switch msg.Type {
		case "search":
			go s.handleSearch(msg)
		case "get_app":
			go s.handleGetApp(msg)
		case "get_popular":
			go s.handleGetPopular(msg)
		case "install":
			go s.handleAction(msg, true)
		case "remove":
			go s.handleAction(msg, false)
		default:
			s.send(errorOut{Type: "error", ID: msg.ID, Message: "unknown message type: " + msg.Type})
		}
	}
	if err := sc.Err(); err != nil {
		log.Printf("ipc: stdin: %v", err)
	}
}

// handleSearch runs the two-phase search. Starting a new search cancels the
// previous one's in-flight network (AUR) request.
func (s *server) handleSearch(msg inbound) {
	s.curMu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.curMu.Unlock()
	defer cancel()

	s.send(searchOut{
		Type:    "search_results",
		ID:      msg.ID,
		Phase:   "local",
		Results: s.orch.Local(msg.Query),
	})

	complete := s.orch.Complete(ctx, msg.Query)
	if ctx.Err() != nil {
		return // superseded by a newer search
	}
	s.send(searchOut{
		Type:    "search_results",
		ID:      msg.ID,
		Phase:   "complete",
		Results: complete,
	})
}

func (s *server) handleGetApp(msg inbound) {
	app, err := s.index.Get(msg.AppID)
	if err != nil {
		s.send(errorOut{Type: "error", ID: msg.ID, Message: err.Error()})
		return
	}
	s.send(appDetailOut{Type: "app_detail", ID: msg.ID, App: app})
}

// handleAction installs or removes a package via the chosen source. After a
// successful action the registry is rebuilt in the background so the installed
// snapshot picks up the change.
func (s *server) handleAction(msg inbound, install bool) {
	src, ok := s.srcs[msg.SourceType]
	if !ok {
		s.send(errorOut{Type: "error", ID: msg.ID, Message: "unknown source: " + msg.SourceType})
		return
	}
	if msg.PackageName == "" {
		s.send(errorOut{Type: "error", ID: msg.ID, Message: "missing package_name"})
		return
	}
	ctx := context.Background()
	var err error
	if install {
		err = src.Install(ctx, msg.PackageName)
	} else {
		err = src.Remove(ctx, msg.PackageName)
	}
	if err != nil {
		s.send(errorOut{Type: "error", ID: msg.ID, Message: err.Error()})
		return
	}
	// Refresh installed flags synchronously so the next get_app returns the
	// new state. A full rebuild (catalogs versions/icons) still runs in the
	// background.
	s.index.RefreshInstalled(context.Background())
	go s.index.Build(context.Background())
	typ := "install_result"
	if !install {
		typ = "remove_result"
	}
	s.send(actionOut{Type: typ, ID: msg.ID, OK: true})
}

func (s *server) handleGetPopular(msg inbound) {
	results := s.index.Popular()
	if results == nil {
		results = []models.AppEntry{}
	}
	s.send(popularOut{Type: "popular_results", ID: msg.ID, Results: results})
}
