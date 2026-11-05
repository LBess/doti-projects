package refresh

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/LBess/doti-projects/server/internal/project"
	"github.com/LBess/doti-projects/server/internal/store"
)

func TestCheckScrape(t *testing.T) {
	tests := []struct {
		name            string
		scraped, active int
		wantErr         bool
	}{
		{"first run accepts any non-empty scrape", 5, 0, false},
		{"empty scrape is rejected on first run", 0, 0, true},
		{"empty scrape is rejected", 0, 30, true},
		{"exactly half is accepted", 15, 30, false},
		{"under half is rejected", 14, 30, true},
		{"growth is accepted", 40, 30, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkScrape(tt.scraped, tt.active); (err != nil) != tt.wantErr {
				t.Errorf("checkScrape(%d, %d) error = %v, wantErr %v", tt.scraped, tt.active, err, tt.wantErr)
			}
		})
	}
}

type fakeSource struct {
	projects []project.Project
	err      error
}

func (f fakeSource) Fetch(context.Context) ([]project.Project, error) { return f.projects, f.err }

type fakeStore struct {
	active     int
	reconciled int
	city       string
	startedAt  time.Time
	pending    []project.Project
	placements map[string]project.Placement
}

func (f *fakeStore) CountActive(context.Context, string) (int, error) { return f.active, nil }
func (f *fakeStore) Reconcile(_ context.Context, city string, at time.Time, scraped []project.Project) (store.ReconcileResult, error) {
	f.reconciled++
	f.city = city
	f.startedAt = at
	return store.ReconcileResult{}, nil
}

func (f *fakeStore) ListPending(context.Context, string) ([]project.Project, error) {
	return f.pending, nil
}
func (f *fakeStore) SetPlacement(_ context.Context, _, key string, p project.Placement) error {
	if f.placements == nil {
		f.placements = map[string]project.Placement{}
	}
	f.placements[key] = p
	return nil
}

type fakeLocator struct {
	failFor string // source key whose Locate returns an error
	calls   int
}

func (f *fakeLocator) Locate(_ context.Context, p project.Project) (project.Placement, error) {
	f.calls++
	if p.SourceKey == f.failFor {
		return project.Placement{}, errors.New("quota")
	}
	return project.Placement{Status: project.NoLocationFound}, nil
}

func TestRefresh(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	three := []project.Project{{SourceKey: "a"}, {SourceKey: "b"}, {SourceKey: "c"}}
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("trusted scrape is reconciled with the run start time", func(t *testing.T) {
		st := &fakeStore{active: 3}
		r := New("other", fakeSource{projects: three}, st, nil, log)
		r.now = func() time.Time { return fixed }
		if err := r.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
		if st.city != "other" {
			t.Errorf("reconciled city = %q, want other", st.city)
		}
		if st.reconciled != 1 || !st.startedAt.Equal(fixed) {
			t.Errorf("reconciled = %d at %v", st.reconciled, st.startedAt)
		}
	})

	t.Run("suspicious scrape leaves the database alone", func(t *testing.T) {
		st := &fakeStore{active: 30}
		r := New("other", fakeSource{projects: three}, st, nil, log)
		if err := r.Refresh(context.Background()); err == nil {
			t.Fatal("expected an error")
		}
		if st.reconciled != 0 {
			t.Error("reconcile ran despite a rejected scrape")
		}
	})

	t.Run("fetch failure leaves the database alone", func(t *testing.T) {
		st := &fakeStore{active: 3}
		r := New("other", fakeSource{err: errors.New("boom")}, st, nil, log)
		if err := r.Refresh(context.Background()); err == nil {
			t.Fatal("expected an error")
		}
		if st.reconciled != 0 {
			t.Error("reconcile ran despite a failed fetch")
		}
	})
}

func TestRefreshLocatesPending(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	three := []project.Project{{SourceKey: "a"}, {SourceKey: "b"}, {SourceKey: "c"}}

	t.Run("places pending projects, leaving failures pending", func(t *testing.T) {
		st := &fakeStore{active: 3, pending: three}
		loc := &fakeLocator{failFor: "b"}
		r := New("other", fakeSource{projects: three}, st, loc, log)
		if err := r.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
		if loc.calls != 3 {
			t.Errorf("locate calls = %d, want 3", loc.calls)
		}
		if _, ok := st.placements["b"]; ok || len(st.placements) != 2 {
			t.Errorf("placements = %v, want a and c only", st.placements)
		}
	})

	t.Run("no locator means projects are stored but not placed", func(t *testing.T) {
		st := &fakeStore{active: 3, pending: three}
		r := New("other", fakeSource{projects: three}, st, nil, log)
		if err := r.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(st.placements) != 0 {
			t.Errorf("placements = %v", st.placements)
		}
	})

	t.Run("a rejected scrape does not locate anything", func(t *testing.T) {
		st := &fakeStore{active: 30, pending: three}
		loc := &fakeLocator{}
		r := New("other", fakeSource{projects: three}, st, loc, log)
		if err := r.Refresh(context.Background()); err == nil {
			t.Fatal("expected an error")
		}
		if loc.calls != 0 {
			t.Errorf("locate calls = %d, want 0", loc.calls)
		}
	})
}
