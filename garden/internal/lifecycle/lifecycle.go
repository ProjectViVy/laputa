package lifecycle

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dashimaki/garden/internal/server"
	"github.com/dashimaki/garden/internal/supervision"
)

const defaultShutdownTimeout = 30 * time.Second

// Run wires signal handling, supervision, and graceful shutdown for srv.
func Run(parent context.Context, srv *server.Server) error {
	logFile, err := SetupLogging()
	if err != nil {
		log.Printf("lifecycle: file logging disabled: %v", err)
	} else {
		defer logFile.Close()
	}

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sup := supervision.New(
		srv,
		srv.Addr,
		supervision.WithCancel(cancel),
		supervision.WithLogger(log.Default()),
	)
	sup.Start(ctx)

	log.Printf("lifecycle: garden running on %s", srv.Addr)
	<-ctx.Done()
	log.Printf("lifecycle: shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer shutdownCancel()
	return sup.Stop(shutdownCtx)
}

// SetupLogging appends garden logs to ~/.garden/garden.log while keeping stderr.
func SetupLogging() (*os.File, error) {
	dir, err := gardenLogDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(dir, "garden.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.Printf("lifecycle: logging to %s", logPath)
	return f, nil
}

func gardenLogDir() (string, error) {
	if dir := os.Getenv("GARDEN_LOG_DIR"); dir != "" {
		return expandHome(dir)
	}
	return expandHome("~/.garden")
}

func expandHome(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
