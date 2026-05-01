package log

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
)

type Logger struct {
	logger *slog.Logger
	file   *os.File
}

func New(levelStr string, logFile string) (*Logger, error) {
	var level slog.Level
	switch levelStr {
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

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler = slog.NewTextHandler(os.Stderr, opts)

	l := &Logger{}

	if logFile != "" {
		dir := filepath.Dir(logFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		l.file = f
		fileHandler := slog.NewJSONHandler(f, opts)
		handler = newMultiHandler(handler, fileHandler)
	}

	l.logger = slog.New(handler)
	return l, nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Debug(msg string, args ...any) { l.logger.Debug(msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.logger.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.logger.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.logger.Error(msg, args...) }

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
