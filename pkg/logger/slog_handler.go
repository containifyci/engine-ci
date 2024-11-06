package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
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

func New(out io.Writer, level slog.Leveler) *SimpleHandler {
	h := &SimpleHandler{out: out, mu: &sync.Mutex{}}
	// if opts != nil {
	// 	h.opts = *opts
	// }
	// if h.opts.Level == nil {
	// 	h.opts.Level = slog.LevelInfo
	// }
	h.opts.Level = level
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
	// if !r.Time.IsZero() {
	// 	buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time), 0)
	// }
	// buf = h.appendAttr(buf, slog.Any(slog.LevelKey, r.Level))
	// if r.PC != 0 {
	// 	fs := runtime.CallersFrames([]uintptr{r.PC})
	// 	f, _ := fs.Next()
	// 	buf = h.appendAttr(buf, slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", f.File, f.Line)))
	// }
	// buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message))
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
