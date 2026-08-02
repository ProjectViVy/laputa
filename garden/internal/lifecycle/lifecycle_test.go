package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dashimaki/garden/internal/crud"
	"github.com/dashimaki/garden/internal/server"
)

func TestSetupLoggingCreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GARDEN_LOG_DIR", dir)

	f, err := SetupLogging()
	if err != nil {
		t.Fatalf("SetupLogging() error = %v", err)
	}
	defer f.Close()

	logPath := filepath.Join(dir, "garden.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("stat log file: %v", err)
	}
}

func TestRunGracefulShutdown(t *testing.T) {
	h := crud.NewHandler(nil, nil)
	srv := &server.Server{Handler: h, Addr: ":0"}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, srv)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}
