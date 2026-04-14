package server

import (
	"context"
	"github.com/Crowley723/conduit/internal/config"
	"log/slog"
	"os"
	"runtime/debug"
)

func setupLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handlers []slog.Handler

	if cfg.Log.Format == "json" {
		handlers = append(handlers, slog.NewJSONHandler(os.Stderr, opts))
	} else {
		handlers = append(handlers, slog.NewTextHandler(os.Stderr, opts))
	}

	multiHandler := NewMultiHandler(handlers...)
	//stackHandler := NewStackTraceHandler(multiHandler)

	return slog.New(multiHandler)
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
