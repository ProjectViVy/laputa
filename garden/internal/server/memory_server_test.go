package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemoryEndpointsUnavailableWithoutFacade(t *testing.T) {
	srv := &Server{}
	requests := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/v2/memories", bytes.NewBufferString(`{"content":"x","kind":"note"}`)),
		httptest.NewRequest(http.MethodGet, "/v2/memories/mem_1", nil),
		httptest.NewRequest(http.MethodGet, "/v2/memories", nil),
		httptest.NewRequest(http.MethodPatch, "/v2/memories/mem_1", bytes.NewBufferString(`{"content":"y"}`)),
		httptest.NewRequest(http.MethodDelete, "/v2/memories/mem_1", nil),
	}
	for _, req := range requests {
		rec := httptest.NewRecorder()
		srv.HTTPHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s code=%d", req.Method, req.URL.Path, rec.Code)
		}
	}
}
