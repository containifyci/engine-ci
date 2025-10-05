package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/moby/term"
)

const (
	altEnter   = "\x1b[?1049h\x1b[H" // switch to alt buffer, go Home
	altExit    = "\x1b[?1049l"       // leave alt buffer
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	homeClear  = "\x1b[H\x1b[2J" // go Home + clear screen
	eraseLine  = "\x1b[2K"
)

type AltScreen struct {
	w         io.Writer
	lastFrame []string
	enabled   bool
}

func NewAlt(out io.Writer) *AltScreen { return &AltScreen{w: out} }

func (a *AltScreen) Enter() {
	if !isTTY(a.w) {
		return
	}
	enableWindowsVT()
	fmt.Fprint(a.w, altEnter, hideCursor)
	a.enabled = true
	// Ensure cleanup on SIGINT/SIGTERM
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() { <-ch; a.Exit(); os.Exit(1) }()
}

func (a *AltScreen) Exit() {
	if !a.enabled {
		return
	}
	fmt.Fprint(a.w, showCursor, altExit)
	if len(a.lastFrame) > 0 {
		// Print a clean copy (no cursor tricks) so it stays in scrollback
		for _, ln := range a.lastFrame {
			fmt.Fprintln(a.w, ln)
		}
	}
	fmt.Fprint(a.w, showCursor)
	a.enabled = false
}

func (a *AltScreen) Render(lines []string) {
	if !a.enabled {
		return
	}
	a.lastFrame = append(a.lastFrame[:0], lines...)
	var b bytes.Buffer
	b.WriteString(homeClear)
	for _, ln := range lines {
		b.WriteString(eraseLine)
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	_, _ = a.w.Write(b.Bytes())
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(f.Fd())
}

// Minimal VT enabling on Windows terminals
func enableWindowsVT() {
	if runtime.GOOS != "windows" {
		return
	}
	// Safe no-op if unsupported; keep it short for brevity
	// Use golang.org/x/sys/windows if you want to set ENABLE_VIRTUAL_TERMINAL_PROCESSING
}
