package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReplayOutboxEventsByFilterDefaultsLimit(t *testing.T) {
	inputs := []int{0, -1, 101}
	for _, input := range inputs {
		got := normalizeReplayLimit(input)
		if got != 25 {
			t.Fatalf("normalizeReplayLimit(%d) = %d, want 25", input, got)
		}
	}
}

func TestReplayOutboxEventsByFilterKeepsValidLimit(t *testing.T) {
	if got := normalizeReplayLimit(10); got != 10 {
		t.Fatalf("normalizeReplayLimit(10) = %d, want 10", got)
	}
}

func TestCompatibilityDeprecationMiddlewareSetsHeaders(t *testing.T) {
	handler := compatibilityDeprecationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/persons", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("Deprecation header = %q, want true", got)
	}
	if got := rec.Header().Get("X-API-Compatibility"); got != "legacy" {
		t.Fatalf("X-API-Compatibility header = %q, want legacy", got)
	}
	if got := rec.Header().Get("Sunset"); got == "" {
		t.Fatal("Sunset header is empty")
	}
}

func TestLegacyAPIEnabledDefaultsToTrue(t *testing.T) {
	t.Setenv("ENABLE_LEGACY_API", "")

	if !legacyAPIEnabled() {
		t.Fatal("legacyAPIEnabled() = false, want true by default")
	}
}

func TestLegacyAPIEnabledCanBeDisabled(t *testing.T) {
	t.Setenv("ENABLE_LEGACY_API", "false")

	if legacyAPIEnabled() {
		t.Fatal("legacyAPIEnabled() = true, want false")
	}
}

func TestNewRouterWithOptionsDisablesLegacyRoutes(t *testing.T) {
	router := newRouterWithOptions(false)

	req := httptest.NewRequest(http.MethodGet, "/persons", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /persons status = %d, want 404 when legacy API is disabled", rec.Code)
	}
}

func TestNewRouterWithOptionsKeepsLegacyRoutesWhenEnabled(t *testing.T) {
	router := newRouterWithOptions(true)

	req := httptest.NewRequest(http.MethodGet, "/persons", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /persons status = %d, want 401 when legacy API is enabled without auth", rec.Code)
	}
}

func TestUIRoutesServeStaticAssets(t *testing.T) {
	router := newRouterWithOptions(false)

	appReq := httptest.NewRequest(http.MethodGet, "/app", nil)
	appRec := httptest.NewRecorder()
	router.ServeHTTP(appRec, appReq)

	if appRec.Code != http.StatusOK {
		t.Fatalf("GET /app status = %d, want 200", appRec.Code)
	}
	if got := appRec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("GET /app content-type = %q, want text/html", got)
	}
	if !strings.Contains(appRec.Body.String(), `/styles.css`) {
		t.Fatal("GET /app did not include stylesheet reference")
	}

	cssReq := httptest.NewRequest(http.MethodGet, "/styles.css", nil)
	cssRec := httptest.NewRecorder()
	router.ServeHTTP(cssRec, cssReq)

	if cssRec.Code != http.StatusOK {
		t.Fatalf("GET /styles.css status = %d, want 200", cssRec.Code)
	}
	if got := cssRec.Header().Get("Content-Type"); !strings.Contains(got, "text/css") {
		t.Fatalf("GET /styles.css content-type = %q, want text/css", got)
	}
	if got := cssRec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Fatalf("GET /styles.css cache-control = %q, want no-store", got)
	}
	if !strings.Contains(cssRec.Body.String(), ":root") {
		t.Fatal("GET /styles.css did not return stylesheet content")
	}
}
