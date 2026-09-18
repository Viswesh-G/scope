package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Viswesh-G/scope/internal/analysis"
)

func TestHistoryEndpoint(t *testing.T) {
	// Use a temporary directory for history so we don't clobber the user's actual history
	dir := t.TempDir()
	originalWD, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalWD)

	// Save a dummy record
	analysis.Save(analysis.SearchRecord{
		Timestamp:  time.Now().Format(time.RFC3339),
		Pattern:    "test_pattern",
		Path:       ".",
		Workers:    2,
		Matches:    42,
		DurationMs: 15.5,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	w := httptest.NewRecorder()

	handleHistoryAPI(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", res.Status)
	}

	var records []analysis.SearchRecord
	if err := json.NewDecoder(res.Body).Decode(&records); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Pattern != "test_pattern" {
		t.Errorf("expected pattern 'test_pattern', got '%s'", records[0].Pattern)
	}
	if records[0].Matches != 42 {
		t.Errorf("expected matches 42, got %d", records[0].Matches)
	}
}

func TestSearchEndpointInvalidPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/search", strings.NewReader("not valid json"))
	w := httptest.NewRecorder()

	handleSearchAPI(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status Bad Request, got %v", res.Status)
	}
}

func TestSearchEndpointMissingPattern(t *testing.T) {
	payload := `{"path": ".", "ignoreCase": true, "workers": 4}`
	req := httptest.NewRequest(http.MethodPost, "/api/search", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleSearchAPI(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status Bad Request, got %v", res.Status)
	}
}

func TestDashboardEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handleDashboard(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", res.Status)
	}

	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected content type text/html, got %s", contentType)
	}
}
