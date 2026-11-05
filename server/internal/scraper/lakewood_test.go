package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestParseLakewood(t *testing.T) {
	f, err := os.Open("testdata/lakewood_cone_zone.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	projects, err := ParseLakewood(f, LakewoodURL)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 9 {
		t.Fatalf("got %d projects, want 9 (the footer card is not a project)", len(projects))
	}

	first := projects[0]
	if first.Title != "Wadsworth and Morrison Road" {
		t.Errorf("title = %q", first.Title)
	}
	if !strings.HasPrefix(first.Description, "Lakewood, in conjunction with the Colorado Department of Transportation") ||
		strings.Contains(first.Description, "\n") {
		t.Errorf("description = %q", first.Description)
	}
	wantURL := "https://www.lakewoodco.gov/Local-Government/Departments/Public-Works/Cone-Zone/Wadsworth-and-Morrison-Road"
	if first.DetailURL == nil || *first.DetailURL != wantURL {
		t.Errorf("detail URL = %v, want %s", first.DetailURL, wantURL)
	}
	if want := "url:lakewoodco.gov/government/departments/public-works/cone-zone/wadsworth-and-morrison-road"; first.SourceKey != want {
		t.Errorf("source key = %q, want %q", first.SourceKey, want)
	}

	keys := map[string]bool{}
	for _, p := range projects {
		if p.Title == "" || p.Description == "" || p.DetailURL == nil || keys[p.SourceKey] {
			t.Errorf("bad or duplicate project: %+v", p)
		}
		keys[p.SourceKey] = true
	}
}

func TestSourceKeyLakewoodSectionNames(t *testing.T) {
	a := "https://www.lakewoodco.gov/Local-Government/Departments/Public-Works/Cone-Zone/North-Dry-Gulch"
	b := "https://www.lakewoodco.gov/Government/Departments/Public-Works/Cone-Zone/North-Dry-Gulch/"
	if ka, kb := sourceKey("x", &a), sourceKey("x", &b); ka != kb {
		t.Errorf("keys differ: %q vs %q", ka, kb)
	}
}

func TestLakewoodFetch(t *testing.T) {
	page, err := os.ReadFile("testdata/lakewood_cone_zone.html")
	if err != nil {
		t.Fatal(err)
	}
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			http.Error(w, "nope", status)
			return
		}
		w.Write(page)
	}))
	defer server.Close()

	f := NewLakewoodFetcher(server.URL + "/Cone-Zone")
	projects, err := f.Fetch(context.Background())
	if err != nil || len(projects) != 9 {
		t.Fatalf("got %d projects, %v", len(projects), err)
	}

	status = http.StatusForbidden
	if _, err := f.Fetch(context.Background()); err == nil {
		t.Error("a 403 should be an error")
	}
}
