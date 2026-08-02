package supervision

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	defaultHealthInterval = 10 * time.Second
	defaultMaxFailures    = 3
	defaultMaxRestarts    = 3
	defaultRestartDelay   = 5 * time.Second
	defaultShutdownWait   = 30 * time.Second
)

// ManagedServer is the HTTP surface supervised by this package.
type ManagedServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

// HealthChecker probes runtime health (defaults to GET /health on the listen addr).
type HealthChecker func(ctx context.Context) error

// Supervisor starts the HTTP server, monitors health, and retries after crashes.
type Supervisor struct {
	srv            ManagedServer
	addr           string
	healthInterval time.Duration
	maxFailures    int
	maxRestarts    int
	restartDelay   time.Duration
	shutdownWait   time.Duration
	healthCheck    HealthChecker
	cancel         context.CancelFunc
	logger         *log.Logger

	mu           sync.Mutex
	failCount    int
	restartCount int
	stopOnce     sync.Once
}

// Option configures a Supervisor.
type Option func(*Supervisor)

// WithCancel registers a cancel func invoked after repeated health or crash failures.
func WithCancel(cancel context.CancelFunc) Option {
	return func(s *Supervisor) {
		s.cancel = cancel
	}
}

// WithHealthChecker overrides the default HTTP health probe.
func WithHealthChecker(check HealthChecker) Option {
	return func(s *Supervisor) {
		s.healthCheck = check
	}
}

// WithLogger sets the logger used for supervision events.
func WithLogger(logger *log.Logger) Option {
	return func(s *Supervisor) {
		s.logger = logger
	}
}

// WithHealthInterval sets the health probe interval.
func WithHealthInterval(interval time.Duration) Option {
	return func(s *Supervisor) {
		if interval > 0 {
			s.healthInterval = interval
		}
	}
}

// WithMaxFailures sets consecutive health failures before shutdown.
func WithMaxFailures(max int) Option {
	return func(s *Supervisor) {
		if max > 0 {
			s.maxFailures = max
		}
	}
}

// WithMaxRestarts sets consecutive crash restarts before shutdown.
func WithMaxRestarts(max int) Option {
	return func(s *Supervisor) {
		if max > 0 {
			s.maxRestarts = max
		}
	}
}

// WithRestartDelay sets the delay between crash restarts.
func WithRestartDelay(delay time.Duration) Option {
	return func(s *Supervisor) {
		if delay > 0 {
			s.restartDelay = delay
		}
	}
}

// New builds a Supervisor for srv listening on addr.
func New(srv ManagedServer, addr string, opts ...Option) *Supervisor {
	s := &Supervisor{
		srv:            srv,
		addr:           addr,
		healthInterval: defaultHealthInterval,
		maxFailures:    defaultMaxFailures,
		maxRestarts:    defaultMaxRestarts,
		restartDelay:   defaultRestartDelay,
		shutdownWait:   defaultShutdownWait,
		healthCheck:    defaultHealthChecker(healthURL(addr)),
		logger:         log.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start launches the HTTP server and health monitor goroutines.
func (s *Supervisor) Start(ctx context.Context) {
	go s.runServerLoop(ctx)
	go s.healthCheckLoop(ctx)
}

// Stop gracefully shuts down the managed server.
func (s *Supervisor) Stop(ctx context.Context) error {
	var err error
	s.stopOnce.Do(func() {
		if ctx == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), s.shutdownWait)
			defer cancel()
		}
		s.logger.Printf("supervision: shutting down server")
		err = s.srv.Shutdown(ctx)
	})
	return err
}

func (s *Supervisor) runServerLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		err := s.srv.ListenAndServe()
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return
		}

		s.logger.Printf("supervision: server exited: %v", err)

		s.mu.Lock()
		s.restartCount++
		restarts := s.restartCount
		s.mu.Unlock()

		if restarts >= s.maxRestarts {
			s.logger.Printf("supervision: max restarts (%d) reached, stopping", s.maxRestarts)
			s.requestStop(ctx)
			return
		}

		s.logger.Printf("supervision: restarting server in %s (attempt %d/%d)", s.restartDelay, restarts, s.maxRestarts)
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.restartDelay):
		}
	}
}

func (s *Supervisor) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(s.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.healthCheck(ctx); err != nil {
				s.mu.Lock()
				s.failCount++
				failures := s.failCount
				s.mu.Unlock()

				s.logger.Printf("supervision: health check failed (%d/%d): %v", failures, s.maxFailures, err)
				if failures >= s.maxFailures {
					s.logger.Printf("supervision: max health failures reached, stopping")
					s.requestStop(ctx)
					return
				}
				continue
			}

			s.mu.Lock()
			s.failCount = 0
			s.mu.Unlock()
		}
	}
}

func (s *Supervisor) requestStop(ctx context.Context) {
	if s.cancel != nil {
		s.cancel()
		return
	}
	_ = s.Stop(ctx)
}

func defaultHealthChecker(url string) HealthChecker {
	client := &http.Client{Timeout: 5 * time.Second}
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode != http.StatusOK {
			return errors.New("health check returned non-200 status")
		}
		return nil
	}
}

func healthURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://127.0.0.1:7373/health"
	}
	if host == "" || host == "0.0.0.0" || host == "[::]" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/health"
}
