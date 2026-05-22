//go:build ignore

// Run is the dev task runner for Sword.
//
// Build once:
//
//	go build -o run run.go
//
// Then:
//
//	./run          – build sidecar + launch tauri dev
//	./run build    – build sidecar only
//	./run check    – go build/vet + tsc (no binary produced)
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	root := mustRepoRoot()

	cmd := "dev"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "dev":
		buildSidecar(root)
		runTauriDev(root)
	case "build":
		buildSidecar(root)
	case "check":
		runCheck(root)
	default:
		fatalf("unknown command %q — valid: dev, build, check", cmd)
	}
}

// ── sidecar build ────────────────────────────────────────────────────────────

func buildSidecar(root string) {
	triple := rustTargetTriple()
	out := filepath.Join(root, "frontend", "src-tauri", "binaries",
		"sword-backend-"+triple)

	step("building Go sidecar → %s", filepath.Base(out))
	run(filepath.Join(root, "backend"), "go", "build", "-o", out, ".")
	ok("sidecar built")
}

// ── tauri dev ────────────────────────────────────────────────────────────────

func runTauriDev(root string) {
	step("starting tauri dev (Ctrl-C to quit)")
	frontend := filepath.Join(root, "frontend")

	cmd := exec.Command("npm", "run", "tauri", "dev")
	cmd.Dir = frontend
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		fatalf("start tauri dev: %v", err)
	}

	// Forward Ctrl-C to the process group so tauri and vite both exit cleanly.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}()

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fatalf("tauri dev: %v", err)
		}
	}
}

// ── check ────────────────────────────────────────────────────────────────────

func runCheck(root string) {
	step("go build ./...")
	run(filepath.Join(root, "backend"), "go", "build", "./...")

	step("go vet ./...")
	run(filepath.Join(root, "backend"), "go", "vet", "./...")

	step("tsc --noEmit")
	run(filepath.Join(root, "frontend"), "npx", "tsc", "--noEmit")

	ok("all checks passed")
}

// ── helpers ──────────────────────────────────────────────────────────────────

func run(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatalf("%s: %v", name, err)
	}
}

// rustTargetTriple asks rustc for the host triple. Falls back to a
// GOOS/GOARCH guess if rustc is not on PATH.
func rustTargetTriple() string {
	out, err := exec.Command("rustc", "-vV").Output()
	if err == nil {
		sc := bufio.NewScanner(bytes.NewReader(out))
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "host:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "host:"))
			}
		}
	}
	// Fallback: construct a plausible triple from Go's runtime constants.
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	return arch + "-unknown-linux-gnu"
}

func mustRepoRoot() string {
	// run.go lives at the repo root, so the executable's parent is the root
	// when built normally. Fall back to the working directory.
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if _, e := os.Stat(filepath.Join(dir, "backend")); e == nil {
			return dir
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// ── output ───────────────────────────────────────────────────────────────────

const (
	bold  = "\033[1m"
	cyan  = "\033[36m"
	green = "\033[32m"
	reset = "\033[0m"
)

func step(format string, a ...any) {
	fmt.Printf(bold+cyan+"▶ "+reset+bold+format+reset+"\n", a...)
}

func ok(msg string) {
	fmt.Printf(bold+green+"✓ "+reset+msg+"\n")
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, bold+"error: "+reset+format+"\n", a...)
	os.Exit(1)
}
