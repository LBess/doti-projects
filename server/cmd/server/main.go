package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LBess/doti-projects/server/internal/api"
	"github.com/LBess/doti-projects/server/internal/city"
	"github.com/LBess/doti-projects/server/internal/geocode"
	"github.com/LBess/doti-projects/server/internal/refresh"
	"github.com/LBess/doti-projects/server/internal/scraper"
	"github.com/LBess/doti-projects/server/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	interval, err := time.ParseDuration(env("REFRESH_INTERVAL", "12h"))
	if err != nil {
		return errors.New("REFRESH_INTERVAL must be a duration like 12h or 30m")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// The server answers from whatever is in the database, so a failed or slow
	// scrape never blocks requests.
	// Without a key projects are still scraped and served, just never placed
	// on the map. (Assign through the interface so a missing key is a nil
	// Locator, not a non-nil interface holding a nil pointer.)
	var locator refresh.Locator
	if key := os.Getenv("GEOCODING_API_KEY"); key != "" {
		locator = &geocode.Locator{Geocoder: geocode.NewClient(key)}
	} else {
		log.Warn("GEOCODING_API_KEY is not set; projects will not be placed on the map")
	}
	// Each city has its own page to scrape and its own refresher, so one city's
	// broken page never holds up (or rejects) another's.
	sources := map[string]refresh.Source{
		city.Denver.ID:   scraper.NewFetcher(env("DOTI_URL", scraper.DefaultURL)),
		city.Lakewood.ID: scraper.NewLakewoodFetcher(env("LAKEWOOD_URL", scraper.LakewoodURL)),
	}
	for _, c := range city.All() {
		go refresh.New(c.ID, sources[c.ID], db, locator, log).Run(ctx, interval)
	}

	srv := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           api.NewHandler(db, log, os.Getenv("STATIC_DIR")),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Info("listening", "addr", srv.Addr, "refresh_interval", interval)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// A wrapper around os.Getenv for returning a default value if the
// environment variables is not set.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
