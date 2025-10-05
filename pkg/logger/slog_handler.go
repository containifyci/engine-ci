package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/dusted-go/logging/prettylog"
)

type SimpleHandler struct {
	opts Options
	// TODO: state for WithGroup and WithAttrs
	mu  *sync.Mutex
	out io.Writer
}

type Options struct {
	Level slog.Leveler
}

func NewRootLog(logOpts slog.HandlerOptions) slog.Handler {
	return slog.NewTextHandler(os.Stdout, &logOpts)
}

func New(progress string, logOpts slog.HandlerOptions) slog.Handler {
	if progress == "progress" {
		return NewSimpleLog(NewLogAggregator(progress), logOpts.Level)
	}
	return NewPrettyLog(progress, logOpts)
}

func NewSimpleLog(out io.Writer, level slog.Leveler) slog.Handler {
	h := &SimpleHandler{out: out, mu: &sync.Mutex{}}
	h.opts.Level = level
	return h
}

func NewPrettyLog(progress string, logOpts slog.HandlerOptions) slog.Handler {
	h := prettylog.New(&logOpts, prettylog.WithDestinationWriter(NewLogAggregator(progress)))
	return h
}

func (h *SimpleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *SimpleHandler) WithGroup(name string) slog.Handler {
	// TODO: implement.
	return h
}

func (h *SimpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// TODO: implement.
	return h
}

func (h *SimpleHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	buf = fmt.Appendf(buf, "%s ", r.Message)
	// TODO: output the Attrs and groups from WithAttrs and WithGroup.
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})
	// buf = append(buf, "---\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *SimpleHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	// Indent 4 spaces per level.
	// buf = fmt.Appendf(buf, "%*s", indentLevel*4, "")
	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values, to make them easy to parse.
		buf = fmt.Appendf(buf, "%s: %q\t", a.Key, a.Value.String())
	case slog.KindTime:
		// Write times in a standard way, without the monotonic time.
		buf = fmt.Appendf(buf, "%s: %s\t", a.Key, a.Value.Time().Format(time.RFC3339Nano))
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		// If the key is non-empty, write it out and indent the rest of the attrs.
		// Otherwise, inline the attrs.
		if a.Key != "" {
			buf = fmt.Appendf(buf, "%s\t", a.Key)
			// indentLevel++
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
	default:
		buf = fmt.Appendf(buf, "%s:%s\t", a.Key, a.Value)
	}
	return buf
}
