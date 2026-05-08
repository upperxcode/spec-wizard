package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	DefaultLogger *slog.Logger
	IsAnalitic    bool
)

func Init(logPath string) error {
	// Verifica modo analítico
	if os.Getenv("LOG_ANALITIC") == "true" {
		IsAnalitic = true
	}
	// Garante que o diretório existe
	dir := filepath.Dir(logPath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo de log: %v", err)
	}

	// Criamos um MultiWriter para logar no arquivo e (opcionalmente) manter o terminal limpo
	// Se quisermos 100% silencioso no terminal, usamos apenas o 'file'
	mw := io.MultiWriter(file)

	handler := slog.NewJSONHandler(mw, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

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
