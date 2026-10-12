//go:build browser_fixture

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// Explicit test-only local server; no fixtures/configuration in a user home.
func TestPersonaBrowserFixture(t *testing.T) {
	info := os.Getenv("GARDEN_BROWSER_INFO")
	if info == "" {
		t.Skip("explicit browser fixture only")
	}
	s, code := installationServer(t)
	var offset atomic.Int64
	for _, id := range []string{"A", "B"} {
		g, err := s.Owner.Subject(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		g.Persona.SetClock(func() time.Time { return time.Now().Add(time.Duration(offset.Load()) * time.Second) })
	}
	stop := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /_fixture/advance", func(w http.ResponseWriter, r *http.Request) { offset.Add(121); w.WriteHeader(204) })
	mux.HandleFunc("POST /_fixture/stop", func(w http.ResponseWriter, r *http.Request) { close(stop); w.WriteHeader(204) })
	mux.Handle("/", s.HTTPHandler())
	srv := httptest.NewServer(mux)
	defer srv.Close()
	raw, _ := json.Marshal(map[string]string{"url": srv.URL, "pairing_code": code})
	if err := os.WriteFile(info, raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("browser fixture ready")
	select {
	case <-stop:
	case <-time.After(10 * time.Minute):
		t.Fatal("browser fixture timed out")
	}
}
