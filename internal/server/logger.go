package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
)

func NewLogger(level, format string) (*slog.Logger, error) {
	var slogLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	var handlers []slog.Handler

	if format == "json" {
		handlers = append(handlers, slog.NewJSONHandler(os.Stderr, opts))
	} else {
		handlers = append(handlers, slog.NewTextHandler(os.Stderr, opts))
	}

	multiHandler := NewMultiHandler(handlers...)

	return slog.New(multiHandler), nil
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if err := h.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return NewMultiHandler(handlers...)
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return NewMultiHandler(handlers...)
}

type StackTraceHandler struct {
	handler slog.Handler
}

func NewStackTraceHandler(h slog.Handler) *StackTraceHandler {
	return &StackTraceHandler{handler: h}
}

func (h *StackTraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *StackTraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		// Clone the record and add stack trace
		r.Add("stack", string(debug.Stack()))
	}
	return h.handler.Handle(ctx, r)
}

func (h *StackTraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &StackTraceHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *StackTraceHandler) WithGroup(name string) slog.Handler {
	return &StackTraceHandler{handler: h.handler.WithGroup(name)}
}
