package location

import (
	"reflect"
	"strings"
	"testing"

	"github.com/LBess/doti-projects/server/internal/project"
)

func wantContains(t *testing.T, queries []string, i int, substrs ...string) {
	t.Helper()
	for _, s := range substrs {
		if !strings.Contains(queries[i], s) {
			t.Errorf("queries[%d] = %q, want it to contain %q", i, queries[i], s)
		}
	}
}

func extract(t *testing.T, kind, title, description string) *Guess {
	t.Helper()
	g := Extract(title, description, DenverNeighborhoods)
	if g == nil || g.Kind != kind {
		t.Fatalf("Extract(%q) = %+v, want kind %q", title, g, kind)
	}
	return g
}

func TestExtract(t *testing.T) {
	t.Run("explicit from-to corridor sentence", func(t *testing.T) {
		g := extract(t, project.KindCorridor, "1st Avenue Redesign",
			"This project will redesign 1st Avenue from Downing St. to University Blvd., widening the Cherry Creek Trail.")
		wantContains(t, g.Queries, 0, "Downing St", "1st Avenue")
		wantContains(t, g.Queries, 1, "University Blvd", "1st Avenue")
	})

	t.Run("intersection from the title (X and Y)", func(t *testing.T) {
		g := extract(t, project.KindPoint, "8th Avenue and Bannock Street Bridge Repairs",
			"Work will repair the North Bannock Street, 8th Avenue, and South Bannock Street bridges over Cherry Creek at Speer Boulevard.")
		wantContains(t, g.Queries, 0, "8th Avenue", "Bannock Street")
	})

	t.Run("title corridor when bounds appear without from", func(t *testing.T) {
		g := extract(t, project.KindCorridor, "Peoria Street Multimodal Improvements",
			"Improvements along Peoria Street. E 37th Avenue to E 56th Avenue.")
		wantContains(t, g.Queries, 0, "Peoria Street", "37th Avenue")
		wantContains(t, g.Queries, 1, "56th Avenue")
	})

	t.Run("numbered-street range sharing one suffix", func(t *testing.T) {
		g := extract(t, project.KindCorridor, "5280 Trail Development - Acoma Street Segment",
			"This segment of the trail runs along Acoma Street from 10th to 12th Avenues.")
		wantContains(t, g.Queries, 0, "Acoma Street", "10th Avenue")
		wantContains(t, g.Queries, 1, "12th Avenue")
	})

	t.Run("corridor between A and B with a borrowed suffix", func(t *testing.T) {
		g := extract(t, project.KindCorridor, "Santa Fe Streetscape Improvements",
			"Denver's DOTI is transforming a stretch of Santa Fe Drive between 6th and Colfax Avenues into a safer street.")
		wantContains(t, g.Queries, 0, "Santa Fe Drive", "6th Avenue")
		wantContains(t, g.Queries, 1, "Colfax Avenue")
	})

	t.Run("Broadway (a suffix-less street) is a valid bound", func(t *testing.T) {
		g := extract(t, project.KindCorridor, "17th and 18th Avenues Bus Priority & Vision Zero Project",
			"Improve pedestrian safety along 17th and 18th Avenues between Broadway and Colorado Boulevard.")
		wantContains(t, g.Queries, 0, "Broadway")
		wantContains(t, g.Queries, 1, "Colorado Boulevard")
		// Must NOT be the nonsensical self-referential "18th Avenues: 17th to 18th".
		if strings.Contains(g.Label, "17th Avenue to 18th Avenue") {
			t.Errorf("label = %q", g.Label)
		}
	})

	t.Run("bridge over road", func(t *testing.T) {
		g := extract(t, project.KindPoint, "Quebec Street Bridge over Airlawn Road",
			"This project rehabilitates the Quebec Street bridge, located south of I-70.")
		wantContains(t, g.Queries, 0, "Quebec Street", "Airlawn Road")
	})

	t.Run("city-wide project with no specific street location", func(t *testing.T) {
		if g := Extract("Neighborhood Intersection Improvements Project",
			"This citywide program identifies and improves intersections across many Denver neighborhoods based on safety data.", DenverNeighborhoods); g != nil {
			t.Errorf("got %+v, want nil", g)
		}
	})

	t.Run("plan or program with no location signal", func(t *testing.T) {
		if g := Extract("Denver Moves Everyone 2050",
			"The citywide transportation plan that guides investments across Denver over the next several decades.", DenverNeighborhoods); g != nil {
			t.Errorf("got %+v, want nil", g)
		}
	})

	t.Run("single named neighborhood", func(t *testing.T) {
		g := extract(t, project.KindNeighborhood, "GES Safer Streets",
			"This project will study and implement safety improvements in the Globeville neighborhood.")
		if want := []string{"Globeville"}; !reflect.DeepEqual(g.Queries, want) {
			t.Errorf("queries = %v, want %v", g.Queries, want)
		}
	})

	t.Run("multiple adjacent neighborhoods", func(t *testing.T) {
		g := extract(t, project.KindNeighborhood, "Globeville Elyria-Swansea (GES) Program of Projects",
			"A collection of improvements across the Globeville and Elyria-Swansea neighborhoods.")
		if want := []string{"Globeville", "Elyria-Swansea"}; !reflect.DeepEqual(g.Queries, want) {
			t.Errorf("queries = %v, want %v", g.Queries, want)
		}
	})

	t.Run("street match is preferred over a neighborhood mention", func(t *testing.T) {
		extract(t, project.KindCorridor, "1st Avenue Redesign",
			"This project will redesign 1st Avenue from Downing St. to University Blvd., passing through the Cherry Creek neighborhood.")
	})

	t.Run("Lane in a project title is a project word", func(t *testing.T) {
		got := Extract("Alameda Lane Repurposing Demonstration Project – Lincoln Street to Franklin Street",
			"The Alameda Lane Repurposing Demonstration Project aims to improve safety along the Alameda Avenue corridor.", DenverNeighborhoods)
		want := &Guess{
			Kind: project.KindCorridor,
			Queries: []string{
				"Lincoln Street & Alameda Avenue",
				"Franklin Street & Alameda Avenue",
			},
			Label: "Alameda Avenue: Lincoln Street to Franklin Street",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("corridor bounded by A to the South and B to the North", func(t *testing.T) {
		got := Extract("Federal Boulevard Improvements",
			"Improvements to enhance the Federal Blvd. corridor bounded by Floyd Ave. to the South and Columbine Rd. to the North.", DenverNeighborhoods)
		want := &Guess{
			Kind: project.KindCorridor,
			Queries: []string{
				"Floyd Ave & Federal Boulevard",
				"Columbine Rd & Federal Boulevard",
			},
			Label: "Federal Boulevard: Floyd Ave to Columbine Rd",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func TestExtractTitleStreet(t *testing.T) {
	t.Run("single street named in the title", func(t *testing.T) {
		g := extract(t, project.KindPoint, "Welton Next Steps",
			"Welton Next Steps is a planning effort to establish a clear vision for the future of Welton Street.")
		wantContains(t, g.Queries, 0, "Welton Street")
	})

	t.Run("street mentioned only in the description is ignored", func(t *testing.T) {
		if g := Extract("Bike Network Study", "Builds on recent work along Colfax Avenue.", DenverNeighborhoods); g != nil {
			t.Errorf("got %+v, want nil", g)
		}
	})

	t.Run("a neighborhood still wins over the title street", func(t *testing.T) {
		g := extract(t, project.KindNeighborhood, "Welton Next Steps",
			"A vision for Welton Street in the Five Points neighborhood.")
		wantContains(t, g.Queries, 0, "Five Points")
	})
}

func TestFindNeighborhoods(t *testing.T) {
	tests := []struct {
		name, text string
		want       []string
	}{
		{"single", "Safety improvements in the Westwood neighborhood.", []string{"Westwood"}},
		{"multiple, in order of first mention, deduped",
			"Covers the Globeville and Elyria-Swansea neighborhoods, with Globeville seeing the most work.",
			[]string{"Globeville", "Elyria-Swansea"}},
		{"prefers North Park Hill over Park Hill", "Improvements in North Park Hill.", []string{"North Park Hill"}},
		{"Park Hill alone", "Improvements in Park Hill.", []string{"Park Hill"}},
		{"prefers City Park West over City Park", "Improvements in City Park West.", []string{"City Park West"}},
		{"University Blvd is not the neighborhood", "Safety work along University Blvd.", nil},
		{"University of Denver is not the neighborhood", "Near the University of Denver campus.", nil},
		{"University Safer Streets is not the neighborhood", "Part of the University Safer Streets Program.", nil},
		{"University neighborhood", "Improvements in the University neighborhood.", []string{"University"}},
		{"Cherry Creek trail is not a neighborhood", "This project will widen the Cherry Creek Trail.", nil},
		{"Cherry Creek crossing is not a neighborhood", "Critical connections across Cherry Creek.", nil},
		{"University Park", "Work in University Park.", []string{"University Park"}},
		{"University Hills", "Work in University Hills.", []string{"University Hills"}},
		{"South Platte River is not a neighborhood", "A study of the South Platte River corridor.", nil},
		{"College View", "Improvements in College View.", []string{"College View/South Platte"}},
		{"none", "A citywide program with no specific location.", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DenverNeighborhoods.find(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DenverNeighborhoods.find(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestExtractWithoutNeighborhoods(t *testing.T) {
	// A city with no neighborhood list never falls back to Denver's.
	title, desc := "Globeville Safety Study", "This project will study and implement safety improvements in the Globeville neighborhood."
	if g := Extract(title, desc, nil); g != nil {
		t.Errorf("got %+v, want nil", g)
	}
	if g := Extract(title, desc, DenverNeighborhoods); g == nil || g.Kind != project.KindNeighborhood {
		t.Errorf("with Denver's neighborhoods got %+v, want a neighborhood guess", g)
	}
}
