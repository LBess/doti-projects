package scraper

import (
	"os"
	"strings"
	"testing"
)

func TestParseFixture(t *testing.T) {
	f, err := os.Open("testdata/projects.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	projects, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	// Snapshot of the live page; update when the fixture is refreshed.
	if len(projects) != 59 {
		t.Errorf("got %d projects, want 59", len(projects))
	}

	keys := map[string]bool{}
	byTitle := map[string]int{}
	for i, p := range projects {
		if keys[p.SourceKey] {
			t.Errorf("duplicate source key %q", p.SourceKey)
		}
		keys[p.SourceKey] = true
		byTitle[p.Title] = i
		if p.Description == "" {
			t.Errorf("%q has an empty description", p.Title)
		}
	}

	// First and last Transportation/Mobility panels bound the section.
	if projects[0].Title != "1st Avenue Redesign" {
		t.Errorf("first project = %q", projects[0].Title)
	}
	if last := projects[len(projects)-1].Title; last != "Welton Next Steps" {
		t.Errorf("last project = %q", last)
	}

	for _, excluded := range []string{
		"Annual Street Paving, Concrete Program & ADA Ramp Construction", // Public Infrastructure
		"Lift Station 13", // Storm/Sanitary Sewer Infrastructure
		"Valverde OSP",    // follows the panel with embedded <h2>s
		"Weir Gulch",      // ditto
	} {
		if _, ok := byTitle[excluded]; ok {
			t.Errorf("%q should have been filtered out", excluded)
		}
	}

	// Relative hrefs are resolved against the site origin.
	first := projects[0]
	if first.DetailURL == nil || !strings.HasPrefix(*first.DetailURL, "https://www.denvergov.org/Government/") {
		t.Errorf("first project detail URL = %v", first.DetailURL)
	}

	// Panels with no "Read more" link have no detail URL and fall back to a title key.
	p := projects[byTitle["12th Ave/Speer Blvd/Elati St Project"]]
	if p.DetailURL != nil {
		t.Errorf("expected nil detail URL, got %q", *p.DetailURL)
	}
	if p.SourceKey != "title:12th-ave-speer-blvd-elati-st-project" {
		t.Errorf("title-derived key = %q", p.SourceKey)
	}
}

func TestSourceKey(t *testing.T) {
	ptr := func(s string) *string { return &s }
	const prefixA = "https://denvergov.org/Government/Agencies-Departments-Offices/Agencies-Departments-Offices-Directory/Department-of-Transportation-and-Infrastructure/Programs-Services/"
	const prefixB = "https://www.denvergov.org/Government/Departments/Department-of-Transportation-and-Infrastructure/Programs-Services/"

	tests := []struct {
		name  string
		title string
		url   *string
		want  string
	}{
		{"directory prefix and www do not matter", "x", ptr(prefixA + "Projects/Colfax-BRT"), "url:denvergov.org/programs-services/projects/colfax-brt"},
		{"same project, other prefix", "x", ptr(prefixB + "Projects/Colfax-BRT"), "url:denvergov.org/programs-services/projects/colfax-brt"},
		{"transfer param is dropped", "x", ptr(prefixA + "Projects/Welton-Next-Steps?transfer=3030c300"), "url:denvergov.org/programs-services/projects/welton-next-steps"},
		{"trailing slash is dropped", "x", ptr(prefixA + "Projects/Colfax-BRT/"), "url:denvergov.org/programs-services/projects/colfax-brt"},
		{"fragment is kept", "x", ptr(prefixB + "Bicycles/Networks#section-2"), "url:denvergov.org/programs-services/bicycles/networks#section-2"},
		{"external hosts keep their host", "x", ptr("https://www.flydenver.com/about-den/pena-master-plan/"), "url:flydenver.com/about-den/pena-master-plan"},
		{"no url falls back to title", "Lift Station 13!", nil, "title:lift-station-13"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sourceKey(tt.title, tt.url); got != tt.want {
				t.Errorf("sourceKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
