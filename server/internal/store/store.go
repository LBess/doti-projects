// Package store persists DOTI projects in Postgres.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/LBess/doti-projects/server/internal/project"
)

const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	pool *pgxpool.Pool
}

// Open connects to the database and applies any pending migrations.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// goose runs on database/sql, so give it its own handle on the same pool.
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	if err := migrate(ctx, db); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func (store *Store) Close() { store.pool.Close() }

func (store *Store) Ping(ctx context.Context) error { return store.pool.Ping(ctx) }

// List returns the projects with the given status, oldest first, with where
// each one goes on the map (or why it isn't).
func (store *Store) List(ctx context.Context, status string) ([]project.Project, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT source_key, title, description, detail_url,
		       geocode_status, location_kind, location_label, lat, lng
		FROM projects
		WHERE status = $1
		ORDER BY first_seen_at, source_key`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []project.Project{}
	for rows.Next() {
		var (
			p           project.Project
			placement   string
			kind, label *string
			lat, lng    *float64
		)
		if err := rows.Scan(&p.SourceKey, &p.Title, &p.Description, &p.DetailURL,
			&placement, &kind, &label, &lat, &lng); err != nil {
			return nil, err
		}
		if placement == project.Mapped && kind != nil && label != nil && lat != nil && lng != nil {
			p.Location = &project.Location{Kind: *kind, Label: *label}
			p.Position = &project.Point{Lat: *lat, Lng: *lng}
		} else {
			p.UnmappedReason = &placement
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// ListPending returns the active projects that still need a placement, oldest
// first.
func (store *Store) ListPending(ctx context.Context) ([]project.Project, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT source_key, title, description, detail_url
		FROM projects
		WHERE status = $1 AND geocode_status = $2
		ORDER BY first_seen_at, source_key`, StatusActive, project.Pending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []project.Project{}
	for rows.Next() {
		var p project.Project
		if err := rows.Scan(&p.SourceKey, &p.Title, &p.Description, &p.DetailURL); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// SetPlacement records where a project goes on the map, or why it can't.
func (store *Store) SetPlacement(ctx context.Context, sourceKey string, placement project.Placement) error {
	var kind, label *string
	var lat, lng *float64
	if placement.Location != nil {
		kind, label = &placement.Location.Kind, &placement.Location.Label
	}
	if placement.Position != nil {
		lat, lng = &placement.Position.Lat, &placement.Position.Lng
	}
	_, err := store.pool.Exec(ctx, `
		UPDATE projects
		SET geocode_status = $2, location_kind = $3, location_label = $4, lat = $5, lng = $6
		WHERE source_key = $1`,
		sourceKey, placement.Status, kind, label, lat, lng)
	return err
}

func (store *Store) CountActive(ctx context.Context) (int, error) {
	var n int
	err := store.pool.QueryRow(ctx, `SELECT count(*) FROM projects WHERE status = $1`, StatusActive).Scan(&n)
	return n, err
}

// ReconcileResult summarizes what a Reconcile call changed.
type ReconcileResult struct {
	Added     int // seen for the first time
	Completed int // were active, absent from this scrape
}

// Reconcile brings the table in line with a trusted scrape, in one transaction:
//
//  1. Upsert every scraped project as active, stamping last_seen_at with
//     runStartedAt. This also revives a completed project that reappears,
//     and sends a project whose title or description changed back to pending
//     so its location is worked out again.
//  2. Mark completed every active project whose last_seen_at is older than
//     runStartedAt, i.e. that was not in this scrape.
//
// runStartedAt is stamped from the caller's clock rather than now() so that
// step 2's comparison can't be thrown off by clock skew or by the time the
// upserts take.
func (store *Store) Reconcile(ctx context.Context, runStartedAt time.Time, scraped []project.Project) (ReconcileResult, error) {
	var result ReconcileResult
	runStartedAt = runStartedAt.UTC().Truncate(time.Microsecond) // Postgres timestamptz precision

	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		batch := &pgx.Batch{}
		for _, p := range scraped {
			batch.Queue(`
				INSERT INTO projects (source_key, title, description, detail_url, status, first_seen_at, last_seen_at)
				VALUES ($1, $2, $3, $4, $5, $6, $6)
				ON CONFLICT (source_key) DO UPDATE SET
					title        = EXCLUDED.title,
					description  = EXCLUDED.description,
					detail_url   = EXCLUDED.detail_url,
					status       = EXCLUDED.status,
					last_seen_at = EXCLUDED.last_seen_at,
					geocode_status = CASE
						WHEN projects.title = EXCLUDED.title AND projects.description = EXCLUDED.description
						THEN projects.geocode_status
						ELSE 'pending'
					END
				RETURNING first_seen_at = $6`,
				p.SourceKey, p.Title, p.Description, p.DetailURL, StatusActive, runStartedAt)
		}

		results := tx.SendBatch(ctx, batch)
		for range scraped {
			var inserted bool
			if err := results.QueryRow().Scan(&inserted); err != nil {
				results.Close()
				return fmt.Errorf("upsert project: %w", err)
			}
			if inserted {
				result.Added++
			}
		}
		if err := results.Close(); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			UPDATE projects SET status = $1
			WHERE status = $2 AND last_seen_at < $3`,
			StatusCompleted, StatusActive, runStartedAt)
		if err != nil {
			return fmt.Errorf("complete missing projects: %w", err)
		}
		result.Completed = int(tag.RowsAffected())
		return nil
	})
	return result, err
}
