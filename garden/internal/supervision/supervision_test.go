package supervision

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type stubServer struct {
	listenErr   error
	shutdownErr error
	listens     atomic.Int32
	shutdowns   atomic.Int32
}

func (s *stubServer) ListenAndServe() error {
	s.listens.Add(1)
	return s.listenErr
}

func (s *stubServer) Shutdown(context.Context) error {
	s.shutdowns.Add(1)
	return s.shutdownErr
}

func TestSupervisorStopCallsShutdown(t *testing.T) {
	srv := &stubServer{listenErr: http.ErrServerClosed}
	sup := New(srv, ":7373", WithHealthChecker(alwaysHealthy))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sup.Start(ctx)

	if err := sup.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if srv.shutdowns.Load() != 1 {
		t.Fatalf("shutdowns = %d, want 1", srv.shutdowns.Load())
	}
}

func TestSupervisorHealthFailuresTriggerCancel(t *testing.T) {
	srv := &stubServer{listenErr: http.ErrServerClosed}
	ctx, cancel := context.WithCancel(context.Background())

	sup := New(
		srv,
		":7373",
		WithCancel(cancel),
		WithHealthInterval(20*time.Millisecond),
		WithMaxFailures(3),
		WithHealthChecker(func(context.Context) error {
			return errors.New("unhealthy")
		}),
	)
	sup.Start(ctx)

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after health failures")
	}
}

func TestSupervisorCrashRestartsUntilMax(t *testing.T) {
	srv := &stubServer{listenErr: errors.New("crash")}
	ctx, cancel := context.WithCancel(context.Background())

	sup := New(
		srv,
		":7373",
		WithCancel(cancel),
		WithRestartDelay(10*time.Millisecond),
		WithMaxRestarts(3),
		WithHealthInterval(time.Hour),
	)
	sup.Start(ctx)

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after max restarts")
	}

	if got := srv.listens.Load(); got < 3 {
		t.Fatalf("listens = %d, want at least 3", got)
	}
}

func alwaysHealthy(context.Context) error {
	return nil
}
