package proc

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Known polkit authentication agent binaries, ordered by preference. The
// first one found on PATH gets spawned when no agent is detected.
var agentBinaries = []string{
	"polkit-gnome-authentication-agent-1",
	"lxqt-policykit-agent",
	"polkit-kde-authentication-agent-1",
	"mate-polkit",
	"xfce-polkit",
	"hyprpolkitagent",
}

// Paths checked directly when the binary name isn't on PATH but the typical
// install location is. polkit-gnome's binary lives in libexec and not PATH on
// many distros, which causes the LookPath probe to miss it.
var agentFullPaths = []string{
	"/usr/lib/polkit-gnome/polkit-gnome-authentication-agent-1",
	"/usr/libexec/polkit-gnome-authentication-agent-1",
	"/usr/lib/polkit-kde-authentication-agent-1",
	"/usr/libexec/polkit-kde-authentication-agent-1",
	"/usr/lib/mate-polkit/polkit-mate-authentication-agent-1",
	"/usr/libexec/mate-polkit/polkit-mate-authentication-agent-1",
	"/usr/lib/hyprpolkitagent/hyprpolkitagent",
}

var (
	spawnMu     sync.Mutex
	spawnedOnce bool
)

// EnsurePolkitAgent best-effort guarantees a polkit authentication agent is
// running. Safe to call multiple times; only spawns when no agent is detected
// in the current user's process list.
func EnsurePolkitAgent() {
	spawnMu.Lock()
	defer spawnMu.Unlock()
	if polkitAgentRunning() {
		return
	}
	if spawnAgent() {
		spawnedOnce = true
		// Give the agent a moment to register with polkitd on the system bus.
		time.Sleep(500 * time.Millisecond)
	}
}

// NeedsAgent returns true when an install/remove error looks like polkit
// failing because no authentication agent is registered for this session.
// Triggers a respawn + retry from the caller.
func NeedsAgent(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "textual authentication agent") ||
		strings.Contains(s, "no authentication agent found") ||
		strings.Contains(s, "no session for cookie") ||
		strings.Contains(s, "/dev/tty")
}

func polkitAgentRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Match common agent process names. -f scans the full command line so
	// agents installed under libexec paths still match.
	out, err := exec.CommandContext(ctx,
		"pgrep", "-u", currentUID(), "-f",
		"polkit-(gnome|kde|mate|lxqt|xfce).*[Aa]gent|polkit-1-auth").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func spawnAgent() bool {
	for _, name := range agentBinaries {
		if path, err := exec.LookPath(name); err == nil {
			return startAgent(name, path)
		}
	}
	for _, path := range agentFullPaths {
		if fileExists(path) {
			return startAgent(path, path)
		}
	}
	log.Printf("polkit: no agent running and none of the known agent binaries are installed — install polkit-gnome or your DE's polkit agent")
	return false
}

func startAgent(label, path string) bool {
	cmd := exec.Command(path)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		log.Printf("polkit: failed to start %s: %v", label, err)
		return false
	}
	log.Printf("polkit: spawned %s (pid %d)", label, cmd.Process.Pid)
	// Reap on exit so it doesn't linger as a zombie if the agent dies.
	go func() { _ = cmd.Wait() }()
	return true
}

func fileExists(p string) bool {
	cmd := exec.Command("test", "-x", p)
	return cmd.Run() == nil
}

func currentUID() string {
	out, err := exec.Command("id", "-u").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
