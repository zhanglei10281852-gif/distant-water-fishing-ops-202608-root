package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/clock"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/service"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/storage/sqlite"
	"golang.org/x/crypto/bcrypt"
)

type httpFixture struct {
	server   *httptest.Server
	store    *sqlite.Store
	services *service.Services
	token    string
}

func newHTTPFixture(t *testing.T) *httpFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	fixed := clock.NewFixed(now)
	services := service.New(store, fixed, 4*time.Hour, 30*time.Minute)
	hash, err := bcrypt.GenerateFromPassword([]byte("very-secure-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user := domain.User{ID: "usr_http", Email: "ops@example.test", DisplayName: "HTTP Ops", PasswordHash: string(hash), Role: domain.RoleVoyageCoordinator, Status: domain.UserActive, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertUser(ctx, user) }); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(New(services, store, nil))
	fixture := &httpFixture{server: httpServer, store: store, services: services}
	t.Cleanup(func() { httpServer.Close(); _ = store.Close() })
	return fixture
}

func (f *httpFixture) request(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, f.server.URL+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func readResponse(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	defer response.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestHealthAndReadyEndpoints(t *testing.T) {
	f := newHTTPFixture(t)
	for _, path := range []string{"/healthz", "/readyz"} {
		response := f.request(t, http.MethodGet, path, nil, "")
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d", path, response.StatusCode)
		}
		if response.Header.Get("X-Request-ID") == "" {
			t.Fatalf("%s missing request id", path)
		}
		_ = readResponse(t, response)
	}
}

func TestProtectedEndpointRequiresBearerToken(t *testing.T) {
	f := newHTTPFixture(t)
	response := f.request(t, http.MethodPost, "/api/v1/fishing_permits", map[string]any{}, "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.StatusCode)
	}
	body := readResponse(t, response)
	if body["error"].(map[string]any)["code"] != "unauthenticated" {
		t.Fatalf("body = %+v", body)
	}
}

func TestUnknownBearerTokenIsUnauthenticated(t *testing.T) {
	f := newHTTPFixture(t)
	response := f.request(t, http.MethodGet, "/api/v1/summary", nil, "token-that-does-not-exist")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	body := readResponse(t, response)
	if body["error"].(map[string]any)["code"] != "unauthenticated" {
		t.Fatalf("body = %+v", body)
	}
}

func TestLoginReturnsTokenAndCreatesFishingPermit(t *testing.T) {
	f := newHTTPFixture(t)
	response := f.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": "ops@example.test", "password": "very-secure-password"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", response.StatusCode)
	}
	body := readResponse(t, response)
	token, ok := body["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login body = %+v", body)
	}
	response = f.request(t, http.MethodPost, "/api/v1/fishing_permits", map[string]any{"code": "RLV-HTTP", "name": "Western Pacific tuna program", "minimum_catch_variance_kg": -0.8, "maximum_catch_variance_kg": 0.8, "max_voyage_duration_hours": 12, "review_hours": 4, "business_timezone": "Asia/Shanghai"}, token)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create program status = %d", response.StatusCode)
	}
	created := readResponse(t, response)
	if created["code"] != "RLV-HTTP" || created["status"] != string(domain.FishingPermitDraft) {
		t.Fatalf("created program = %+v", created)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	f := newHTTPFixture(t)
	response := f.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": "ops@example.test", "password": "wrong-password"}, "")
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d", response.StatusCode)
	}
	_ = readResponse(t, response)
}

func TestLogoutRevokesSession(t *testing.T) {
	f := newHTTPFixture(t)
	login := f.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": "ops@example.test", "password": "very-secure-password"}, "")
	token := readResponse(t, login)["token"].(string)

	logout := f.request(t, http.MethodPost, "/api/v1/auth/logout", nil, token)
	if logout.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d", logout.StatusCode)
	}
	_ = logout.Body.Close()

	reused := f.request(t, http.MethodPost, "/api/v1/fishing_permits", map[string]any{}, token)
	if reused.StatusCode == http.StatusCreated || reused.StatusCode == http.StatusOK {
		t.Fatalf("revoked session was accepted with status %d", reused.StatusCode)
	}
	_ = readResponse(t, reused)
}

func TestExpiredSessionIsRejected(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "expired-http.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	fixed := clock.NewFixed(now)
	services := service.New(store, fixed, 15*time.Minute, 30*time.Minute)
	hash, err := bcrypt.GenerateFromPassword([]byte("very-secure-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user := domain.User{ID: "usr_expired", Email: "expired@example.test", DisplayName: "Expired User", PasswordHash: string(hash), Role: domain.RoleVoyageCoordinator, Status: domain.UserActive, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertUser(ctx, user) }); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(New(services, store, nil))
	t.Cleanup(httpServer.Close)
	f := &httpFixture{server: httpServer, store: store, services: services}
	login := f.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": user.Email, "password": "very-secure-password"}, "")
	token := readResponse(t, login)["token"].(string)
	fixed.Advance(16 * time.Minute)

	response := f.request(t, http.MethodPost, "/api/v1/fishing_permits", map[string]any{}, token)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	_ = readResponse(t, response)
}

func TestUnknownJSONFieldAndOversizedBodyAreRejected(t *testing.T) {
	f := newHTTPFixture(t)
	login := f.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": "ops@example.test", "password": "very-secure-password"}, "")
	body := readResponse(t, login)
	token := body["token"].(string)
	unknown := f.request(t, http.MethodPost, "/api/v1/fishing_permits", map[string]any{"code": "X", "unexpected": true}, token)
	if unknown.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d", unknown.StatusCode)
	}
	_ = readResponse(t, unknown)
	large := strings.Repeat("x", 2<<20)
	req, err := http.NewRequest(http.MethodPost, f.server.URL+"/api/v1/fishing_permits", strings.NewReader(`{"code":"`+large+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("large body status = %d", response.StatusCode)
	}
	_ = readResponse(t, response)
}

func TestRequestIDIsPropagated(t *testing.T) {
	f := newHTTPFixture(t)
	req, err := http.NewRequest(http.MethodGet, f.server.URL+"/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-ID", "req-fixed")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("X-Request-ID") != "req-fixed" {
		t.Fatalf("request id = %q", response.Header.Get("X-Request-ID"))
	}
}
