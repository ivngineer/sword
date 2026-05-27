package proc

import (
	"regexp"
	"strconv"
	"strings"
)

// percentRe matches the first occurrence of a percentage in a line, e.g.
// "Installing… 35%", "[####    ] 35.4 %", "Receiving objects: 35% (12/34)".
var percentRe = regexp.MustCompile(`(\d{1,3}(?:\.\d+)?)\s*%`)

// ParseProgress extracts a [0,1] fraction and short status from a raw line of
// stdout/stderr produced by pacman, paru, yay, or flatpak. Returns
// (fraction=-1, status) when no percentage is parseable, signalling
// indeterminate progress with a fresh status label.
//
// The status is a trimmed, length-capped version of the line so the UI can
// show "what is happening right now" without bringing in tool-specific
// formatting noise.
func ParseProgress(line string) (float64, string) {
	status := truncate(strings.TrimSpace(line), 120)
	if status == "" {
		return -1, ""
	}
	m := percentRe.FindStringSubmatch(line)
	if m == nil {
		return -1, status
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return -1, status
	}
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return v / 100.0, status
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
