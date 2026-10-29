package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/LBess/doti-projects/server/internal/project"
)

// Needs a throwaway Postgres: TEST_DATABASE_URL=postgres://... go test ./internal/store
// The test drops and recreates the projects table.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if _, err := s.pool.Exec(ctx, `TRUNCATE projects`); err != nil {
		t.Fatal(err)
	}
	return s
}

func proj(key string) project.Project {
	return project.Project{SourceKey: key, Title: "Title " + key, Description: "Desc " + key}
}

func keys(t *testing.T, s *Store, status string) []string {
	t.Helper()
	list, err := s.List(context.Background(), status)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.SourceKey
	}
	return out
}

func TestReconcileLifecycle(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	t0 := time.Now().Add(-24 * time.Hour) // keep every fake run in the past

	// Run 1: a and b appear.
	res, err := s.Reconcile(ctx, t0, []project.Project{proj("a"), proj("b")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 2 || res.Completed != 0 {
		t.Errorf("run 1 = %+v", res)
	}

	// Run 2: b is gone, c is new.
	res, err = s.Reconcile(ctx, t0.Add(time.Hour), []project.Project{proj("a"), proj("c")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || res.Completed != 1 {
		t.Errorf("run 2 = %+v", res)
	}
	if got := keys(t, s, StatusActive); len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Errorf("active = %v", got)
	}
	if got := keys(t, s, StatusCompleted); len(got) != 1 || got[0] != "b" {
		t.Errorf("completed = %v", got)
	}

	// Run 3: b reappears and is revived, not re-added.
	res, err = s.Reconcile(ctx, t0.Add(2*time.Hour), []project.Project{proj("a"), proj("b"), proj("c")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || res.Completed != 0 {
		t.Errorf("run 3 = %+v", res)
	}
	if got := keys(t, s, StatusCompleted); len(got) != 0 {
		t.Errorf("completed = %v", got)
	}

	// Edits to a project are picked up.
	edited := proj("a")
	edited.Title = "Renamed"
	url := "https://example.com/a"
	edited.DetailURL = &url
	if _, err := s.Reconcile(ctx, t0.Add(3*time.Hour), []project.Project{edited, proj("b"), proj("c")}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List(ctx, StatusActive)
	if list[0].Title != "Renamed" || list[0].DetailURL == nil || *list[0].DetailURL != url {
		t.Errorf("a = %+v", list[0])
	}
}

func TestPlacement(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	t0 := time.Now().Add(-24 * time.Hour)

	if _, err := s.Reconcile(ctx, t0, []project.Project{proj("a"), proj("b")}); err != nil {
		t.Fatal(err)
	}
	pending, err := s.ListPending(ctx)
	if err != nil || len(pending) != 2 {
		t.Fatalf("pending = %v, %v", pending, err)
	}

	mapped := project.Placement{
		Status:   project.Mapped,
		Location: &project.Location{Kind: project.KindPoint, Label: "A & B"},
		Position: &project.Point{Lat: 39.74, Lng: -104.99},
	}
	if err := s.SetPlacement(ctx, "a", mapped); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPlacement(ctx, "b", project.Placement{Status: project.NoLocationFound}); err != nil {
		t.Fatal(err)
	}
	if pending, _ := s.ListPending(ctx); len(pending) != 0 {
		t.Errorf("pending after placing = %v", pending)
	}

	list, err := s.List(ctx, StatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Position == nil || *list[0].Position != *mapped.Position || *list[0].Location != *mapped.Location || list[0].UnmappedReason != nil {
		t.Errorf("mapped project = %+v", list[0])
	}
	if list[1].Position != nil || list[1].Location != nil || list[1].UnmappedReason == nil || *list[1].UnmappedReason != project.NoLocationFound {
		t.Errorf("unmapped project = %+v", list[1])
	}

	// An unchanged project keeps its placement across scrapes; one whose text
	// changed goes back to pending.
	changed := proj("b")
	changed.Description = "something new"
	if _, err := s.Reconcile(ctx, t0.Add(time.Hour), []project.Project{proj("a"), changed}); err != nil {
		t.Fatal(err)
	}
	pending, _ = s.ListPending(ctx)
	if len(pending) != 1 || pending[0].SourceKey != "b" {
		t.Errorf("pending after text change = %v, want just b", pending)
	}
}
