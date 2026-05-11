package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var (
	DefaultLogger *slog.Logger
	IsAnalitic    bool
)

type PipeHandler struct {
	w io.Writer
}

func (h *PipeHandler) Enabled(ctx context.Context, level slog.Level) bool { return true }
func (h *PipeHandler) Handle(ctx context.Context, r slog.Record) error {
	timeStr := r.Time.Format("2006-01-02T15:04:05.000000000-07:00")
	levelStr := r.Level.String()

	// Coleta todos os atributos extras
	attrs := make([]string, 0)
	mode := "system"
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "mode" {
			mode = a.Value.String()
		} else {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))
		}
		return true
	})

	extra := mode
	if len(attrs) > 0 {
		extra = fmt.Sprintf("%s (%s)", mode, strings.Join(attrs, ", "))
	}

	fmt.Fprintf(h.w, "%s | %s | %s | %s\n", timeStr, levelStr, r.Message, extra)
	return nil
}
func (h *PipeHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *PipeHandler) WithGroup(name string) slog.Handler       { return h }

func Init(logPath string) error {
	// Garante que o diretório existe
	dir := filepath.Dir(logPath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo de log: %v", err)
	}

	mw := io.MultiWriter(file)

	var handler slog.Handler
	// Se LOG_ANALITIC for true, usa o formato pipe-separado
	if os.Getenv("LOG_ANALITIC") == "true" {
		IsAnalitic = true
		handler = &PipeHandler{w: mw}
	} else {
		handler = slog.NewJSONHandler(mw, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	DefaultLogger = slog.New(handler)
	slog.SetDefault(DefaultLogger)

	return nil
}

func Info(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Info(msg, args...)
	}
}

func Error(msg string, err error, args ...any) {
	if DefaultLogger != nil {
		finalArgs := append(args, "error", err)
		DefaultLogger.Error(msg, finalArgs...)
	}
}

func Debug(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(msg, args...)
	}
}

// LogTask é um helper para logar eventos de tarefas com contexto rico
func LogTask(taskID, msg string, args ...any) {
	if DefaultLogger != nil {
		finalArgs := append([]any{"task_id", taskID}, args...)
		DefaultLogger.Info(msg, finalArgs...)
	}
}

func WithContext(ctx context.Context) *slog.Logger {
	return DefaultLogger
}
