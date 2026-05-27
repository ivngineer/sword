// Package proc contains shared helpers for spawning privileged install /
// remove commands. The sidecar inherits a controlling TTY from `tauri dev`
// and from terminal launches in general; pkexec (and any helper that calls
// it) will open /dev/tty directly to prompt for a password, which would
// hang the GUI. Detaching with setsid removes the controlling TTY so polkit
// routes the prompt through the session authentication agent (GNOME
// Shell, polkit-kde-agent, lxqt-policykit-agent, etc.) instead.
package proc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// RunDetached runs a command with no controlling terminal and discarded
// stdin, so child processes that would otherwise read from /dev/tty (pkexec)
// defer to the session polkit agent. Stderr is captured and returned as the
// error message on failure.
//
// When the failure looks like polkit couldn't find an auth agent, we spawn
// one if available and retry once.
func RunDetached(ctx context.Context, name string, args ...string) error {
	err := runOnce(ctx, name, args...)
	if err != nil && NeedsAgent(err) {
		EnsurePolkitAgent()
		err = runOnce(ctx, name, args...)
		if err != nil && NeedsAgent(err) {
			return errors.New("polkit authentication agent missing — install polkit-gnome (or your DE's agent) and ensure it autostarts. Original: " + err.Error())
		}
	}
	return err
}

func runOnce(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:     true,
		Foreground: false,
	}
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return err
		}
		return errors.New(msg)
	}
	return nil
}

// RunStreaming runs name with args and streams stdout+stderr line-by-line to
// onLine. Lines are split on either '\n' or '\r' so that carriage-return-only
// progress redraws (common with flatpak/pacman) become discrete events.
//
// Detaching semantics match RunDetached: no controlling terminal, no stdin,
// polkit prompts route to the session agent. On failure the most recent
// stderr content is returned as the error message; the same NeedsAgent
// respawn-and-retry path is applied.
func RunStreaming(ctx context.Context, onLine func(string), name string, args ...string) error {
	err := runStreamOnce(ctx, onLine, name, args...)
	if err != nil && NeedsAgent(err) {
		EnsurePolkitAgent()
		err = runStreamOnce(ctx, onLine, name, args...)
		if err != nil && NeedsAgent(err) {
			return errors.New("polkit authentication agent missing — install polkit-gnome (or your DE's agent) and ensure it autostarts. Original: " + err.Error())
		}
	}
	return err
}

func runStreamOnce(ctx context.Context, onLine func(string), name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Foreground: false}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var errTail bytes.Buffer
	var mu sync.Mutex
	emit := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		mu.Lock()
		onLine(s)
		mu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); scanCRLF(stdout, emit, nil) }()
	go func() { defer wg.Done(); scanCRLF(stderr, emit, &errTail) }()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(errTail.String())
		if msg == "" {
			return err
		}
		return errors.New(msg)
	}
	return nil
}

// scanCRLF reads r and invokes onLine for each segment terminated by '\n' or
// '\r'. If tail is non-nil, every byte read is also copied into it, capped to
// the last 8 KiB, so callers can recover the tail of stderr for error reports.
func scanCRLF(r io.Reader, onLine func(string), tail *bytes.Buffer) {
	buf := make([]byte, 4096)
	var acc bytes.Buffer
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if tail != nil {
				tail.Write(buf[:n])
				if tail.Len() > 8192 {
					b := tail.Bytes()
					tail.Reset()
					tail.Write(b[len(b)-8192:])
				}
			}
			for _, c := range buf[:n] {
				if c == '\n' || c == '\r' {
					if acc.Len() > 0 {
						onLine(acc.String())
						acc.Reset()
					}
				} else {
					acc.WriteByte(c)
				}
			}
		}
		if err != nil {
			break
		}
	}
	if acc.Len() > 0 {
		onLine(acc.String())
	}
}
