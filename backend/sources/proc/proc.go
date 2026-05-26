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
	"os/exec"
	"strings"
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
