package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LBess/doti-projects/server/internal/project"
)

type fakeStore struct {
	byStatus map[string][]project.Project
	err      error
}

func (f fakeStore) List(_ context.Context, status string) ([]project.Project, error) {
	return f.byStatus[status], f.err
}
func (f fakeStore) Ping(context.Context) error { return f.err }

func serve(s Store, target string) *httptest.ResponseRecorder {
	h := NewHandler(s, slog.New(slog.NewTextHandler(io.Discard, nil)), "")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestProjects(t *testing.T) {
	url := "https://example.com/a"
	s := fakeStore{byStatus: map[string][]project.Project{
		"active":    {{SourceKey: "a", Title: "A", Description: "desc", DetailURL: &url}, {SourceKey: "b", Title: "B"}},
		"completed": {{SourceKey: "c", Title: "C"}},
	}}

	t.Run("defaults to active, with the frontend's field names", func(t *testing.T) {
		rec := serve(s, "/api/projects")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		var got []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0]["id"] != "a" || got[0]["detailUrl"] != url || got[1]["detailUrl"] != nil {
			t.Errorf("body = %s", rec.Body)
		}
	})

	t.Run("serves position and location for mapped projects, a reason for the rest", func(t *testing.T) {
		reason := project.Pending
		placed := fakeStore{byStatus: map[string][]project.Project{"active": {
			{
				SourceKey: "m", Title: "M",
				Position: &project.Point{Lat: 39.74, Lng: -104.99},
				Location: &project.Location{Kind: project.KindPoint, Label: "A & B"},
			},
			{SourceKey: "u", Title: "U", UnmappedReason: &reason},
		}}}
		var got []map[string]any
		if err := json.Unmarshal(serve(placed, "/api/projects").Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if pos, _ := got[0]["position"].(map[string]any); pos["lat"] != 39.74 || pos["lng"] != -104.99 {
			t.Errorf("position = %v", got[0]["position"])
		}
		if loc, _ := got[0]["location"].(map[string]any); loc["kind"] != "point" || loc["label"] != "A & B" {
			t.Errorf("location = %v", got[0]["location"])
		}
		if got[0]["unmappedReason"] != nil || got[1]["position"] != nil || got[1]["unmappedReason"] != "pending" {
			t.Errorf("body = %s", serve(placed, "/api/projects").Body)
		}
	})

	t.Run("status=completed", func(t *testing.T) {
		rec := serve(s, "/api/projects?status=completed")
		if !strings.Contains(rec.Body.String(), `"id":"c"`) {
			t.Errorf("body = %s", rec.Body)
		}
	})

	t.Run("empty list is [] not null", func(t *testing.T) {
		rec := serve(fakeStore{byStatus: map[string][]project.Project{"active": {}}}, "/api/projects")
		if strings.TrimSpace(rec.Body.String()) != "[]" {
			t.Errorf("body = %s", rec.Body)
		}
	})

	t.Run("unknown status", func(t *testing.T) {
		if rec := serve(s, "/api/projects?status=bogus"); rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d", rec.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		if rec := serve(fakeStore{err: errors.New("down")}, "/api/projects"); rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d", rec.Code)
		}
	})
}

func TestHealth(t *testing.T) {
	if rec := serve(fakeStore{}, "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("healthy status = %d", rec.Code)
	}
	if rec := serve(fakeStore{err: errors.New("down")}, "/healthz"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("unhealthy status = %d", rec.Code)
	}
}

func TestStaticFrontend(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"index.html": "INDEX", "assets/app.js": "JS"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h := NewHandler(fakeStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)), dir)
	get := func(target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec
	}

	tests := []struct {
		target string
		code   int
		body   string
	}{
		{"/", 200, "INDEX"},
		{"/assets/app.js", 200, "JS"},
		{"/unmapped", 200, "INDEX"},     // a client-side route
		{"/unmapped/", 200, "INDEX"},    // with a trailing slash
		{"/assets/missing.js", 404, ""}, // a missing file is not the app
		{"/api/nope", 404, ""},          // nor is an unknown API route
	}
	for _, tt := range tests {
		rec := get(tt.target)
		if rec.Code != tt.code || (tt.body != "" && rec.Body.String() != tt.body) {
			t.Errorf("GET %s = %d %q, want %d %q", tt.target, rec.Code, rec.Body.String(), tt.code, tt.body)
		}
	}
}

func TestCities(t *testing.T) {
	rec := serve(fakeStore{}, "/api/cities")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got []struct {
		ID       string           `json:"id"`
		Name     string           `json:"name"`
		Color    string           `json:"color"`
		Boundary [][][][2]float64 `json:"boundary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, c := range got {
		ids = append(ids, c.ID)
		if c.Name == "" || c.Color == "" || len(c.Boundary) == 0 || len(c.Boundary[0]) == 0 || len(c.Boundary[0][0]) < 3 {
			t.Errorf("city %q is missing data", c.ID)
		}
	}
	if strings.Join(ids, ",") != "denver,lakewood" {
		t.Errorf("cities = %v", ids)
	}
}
