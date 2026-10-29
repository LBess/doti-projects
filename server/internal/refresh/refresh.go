// Package refresh periodically re-scrapes the DOTI page, reconciles the
// database with it, and works out where any new projects belong on the map.
package refresh

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/LBess/doti-projects/server/internal/project"
	"github.com/LBess/doti-projects/server/internal/store"
)

// minKeepRatio is the smallest share of the current active projects a scrape
// must return to be trusted. A scrape that comes back much smaller is far more
// likely a broken page or parser than DOTI finishing half its projects at
// once, and acting on it would mark them all completed.
const minKeepRatio = 0.5

type Source interface {
	Fetch(ctx context.Context) ([]project.Project, error)
}

type Store interface {
	CountActive(ctx context.Context) (int, error)
	Reconcile(ctx context.Context, runStartedAt time.Time, scraped []project.Project) (store.ReconcileResult, error)
	ListPending(ctx context.Context) ([]project.Project, error)
	SetPlacement(ctx context.Context, sourceKey string, placement project.Placement) error
}

// Locator works out where a project belongs on the map. An error means the
// attempt failed and should be retried on a later refresh.
type Locator interface {
	Locate(ctx context.Context, p project.Project) (project.Placement, error)
}

type Refresher struct {
	Source Source
	Store  Store
	// Locator may be nil, in which case projects are stored but never placed.
	Locator Locator
	Log     *slog.Logger
	now     func() time.Time
}

func New(source Source, store Store, locator Locator, log *slog.Logger) *Refresher {
	return &Refresher{Source: source, Store: store, Locator: locator, Log: log, now: time.Now}
}

// Run refreshes once immediately, then every interval until ctx is cancelled.
func (r *Refresher) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := r.Refresh(ctx); err != nil {
			r.Log.Error("refresh failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Refresh scrapes the page and, if the result is trustworthy, reconciles the
// database with it, then places any projects that don't have a location yet.
// A suspicious scrape is returned as an error and leaves the database
// untouched.
func (refresher *Refresher) Refresh(ctx context.Context) error {
	startedAt := refresher.now()

	projects, err := refresher.Source.Fetch(ctx)
	if err != nil {
		return err
	}
	active, err := refresher.Store.CountActive(ctx)
	if err != nil {
		return fmt.Errorf("count active projects: %w", err)
	}
	if err := checkScrape(len(projects), active); err != nil {
		return err
	}

	result, err := refresher.Store.Reconcile(ctx, startedAt, projects)
	if err != nil {
		return fmt.Errorf("reconcile projects: %w", err)
	}
	refresher.Log.Info("refresh complete",
		"scraped projects", len(projects), "added", result.Added, "completed", result.Completed, "previously_active", active)

	refresher.locate(ctx)
	return nil
}

// locate places every pending project. It runs after the scrape has been
// reconciled, so a slow or failing geocoder never holds up (or loses) the
// project list: a project whose attempt fails stays pending and is retried on
// the next refresh.
func (refresher *Refresher) locate(ctx context.Context) {
	if refresher.Locator == nil {
		return
	}
	pending, err := refresher.Store.ListPending(ctx)
	if err != nil {
		refresher.Log.Error("list pending projects", "error", err)
		return
	}

	placed, failed := 0, 0
	for _, p := range pending {
		if ctx.Err() != nil {
			return
		}
		placement, err := refresher.Locator.Locate(ctx, p)
		if err != nil {
			refresher.Log.Error("locate project", "project", p.SourceKey, "error", err)
			failed++
			continue
		}
		if err := refresher.Store.SetPlacement(ctx, p.SourceKey, placement); err != nil {
			refresher.Log.Error("save placement", "project", p.SourceKey, "error", err)
			failed++
			continue
		}
		placed++
	}
	if len(pending) > 0 {
		refresher.Log.Info("locate complete", "pending", len(pending), "placed", placed, "failed", failed)
	}
}

// checkScrape reports whether a scrape of `projects` can be trusted
// given `active` projects currently in the database. With no active projects
// (a first run) there is no baseline, so any non-empty scrape is accepted.
func checkScrape(projects, active int) error {
	if projects == 0 {
		return fmt.Errorf("scrape found 0 projects; the DOTI page structure may have changed")
	}
	if float64(projects) < float64(active)*minKeepRatio {
		return fmt.Errorf("scrape found %d projects but %d are active (under %.0f%%); refusing to apply it",
			projects, active, minKeepRatio*100)
	}
	return nil
}
