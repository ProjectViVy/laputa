package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

const ownerCookie = "garden_owner"

type Pairing struct {
	mu          sync.Mutex
	code        [32]byte
	codeExpires time.Time
	used        bool
	sessions    map[[32]byte]time.Time
}

func randomCredential() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// The composition root receives the one-use code for its private local
// channel. It is never returned by status routes, put in URLs, or logged.
func NewPairing() (*Pairing, string, error) {
	code, err := randomCredential()
	if err != nil {
		return nil, "", err
	}
	return &Pairing{code: sha256.Sum256([]byte(code)), codeExpires: time.Now().Add(10 * time.Minute), sessions: map[[32]byte]time.Time{}}, code, nil
}
func (p *Pairing) admit(code string) (string, bool) {
	if p == nil {
		return "", false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	provided := sha256.Sum256([]byte(code))
	if p.used || time.Now().After(p.codeExpires) || subtle.ConstantTimeCompare(p.code[:], provided[:]) != 1 {
		return "", false
	}
	token, err := randomCredential()
	if err != nil {
		return "", false
	}
	p.used = true
	p.code = [32]byte{}
	p.sessions[sha256.Sum256([]byte(token))] = time.Now().Add(24 * time.Hour)
	return token, true
}
func (p *Pairing) owner(r *http.Request) bool {
	if p == nil {
		return false
	}
	cookie, err := r.Cookie(ownerCookie)
	if err != nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	hash := sha256.Sum256([]byte(cookie.Value))
	expiry, ok := p.sessions[hash]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(p.sessions, hash)
		return false
	}
	return true
}
func sameOrigin(r *http.Request) bool {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return r.Header.Get("Origin") == scheme+"://"+r.Host
}
func (s *Server) handlePairing(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeErrorWithCode(w, 403, "principal_forbidden", errInvalidCapability)
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(w, r, 1024, &in); err != nil {
		writeRequestError(w, err)
		return
	}
	token, ok := s.Pairing.admit(in.Code)
	if !ok {
		writeErrorWithCode(w, 401, "authentication_required", errAuthenticationRequired)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: ownerCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil, MaxAge: 86400})
	writeJSON(w, 200, map[string]bool{"paired": true})
}
func (s *Server) requireOwner(w http.ResponseWriter, r *http.Request) bool {
	if !s.Pairing.owner(r) {
		if _, err := s.principalForRequest(r); err == nil {
			writeErrorWithCode(w, 403, "principal_forbidden", errInvalidCapability)
		} else {
			writeErrorWithCode(w, 401, "authentication_required", errAuthenticationRequired)
		}
		return false
	}
	if r.Method != "GET" && r.Method != "HEAD" && !sameOrigin(r) {
		writeErrorWithCode(w, 403, "principal_forbidden", errInvalidCapability)
		return false
	}
	return true
}
