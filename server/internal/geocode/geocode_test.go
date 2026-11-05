package geocode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LBess/doti-projects/server/internal/city"
	"github.com/LBess/doti-projects/server/internal/project"
)

func TestClientGeocode(t *testing.T) {
	var gotQuery map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotQuery = map[string]string{"address": q.Get("address"), "bounds": q.Get("bounds"), "region": q.Get("region"), "key": q.Get("key")}
		switch q.Get("address") {
		case "found, Denver, CO":
			w.Write([]byte(`{"status":"OK","results":[{"geometry":{"location":{"lat":39.74,"lng":-104.99}}}]}`))
		case "missing, Denver, CO":
			w.Write([]byte(`{"status":"ZERO_RESULTS","results":[]}`))
		case "quota, Denver, CO":
			w.Write([]byte(`{"status":"OVER_QUERY_LIMIT","error_message":"slow down","results":[]}`))
		default:
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	c := &Client{APIKey: "secret", HTTP: server.Client(), BaseURL: server.URL}
	ctx := context.Background()

	got, err := c.Geocode(ctx, "found", city.Denver)
	if err != nil || got != (project.Point{Lat: 39.74, Lng: -104.99}) {
		t.Fatalf("found = %v, %v", got, err)
	}
	if gotQuery["bounds"] != city.Denver.GeocodeBounds || gotQuery["region"] != "us" || gotQuery["key"] != "secret" {
		t.Errorf("request params = %v", gotQuery)
	}

	if _, err := c.Geocode(ctx, "missing", city.Denver); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing err = %v, want ErrNotFound", err)
	}
	if _, err := c.Geocode(ctx, "quota", city.Denver); err == nil || errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "OVER_QUERY_LIMIT") {
		t.Errorf("quota err = %v, want a retryable error", err)
	}
	if _, err := c.Geocode(ctx, "other", city.Denver); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("http 500 err = %v, want a retryable error", err)
	}
}

func TestClientGeocodeDoesNotLeakKey(t *testing.T) {
	c := &Client{APIKey: "secret", HTTP: &http.Client{}, BaseURL: "http://127.0.0.1:1"}
	_, err := c.Geocode(context.Background(), "anything", city.Denver)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Errorf("err = %v, want a failure that omits the key", err)
	}
}

type fakeGeocoder map[string]error // query -> error; absent queries resolve downtown

func (f fakeGeocoder) Geocode(_ context.Context, query string, _ city.City) (project.Point, error) {
	if err, ok := f[query]; ok {
		return project.Point{}, err
	}
	return project.Point{Lat: 39.7392, Lng: -104.9903}, nil
}

func TestLocate(t *testing.T) {
	ctx := context.Background()
	corridor := project.Project{City: "denver", Title: "1st Avenue Redesign", Description: "This project will redesign 1st Avenue from Downing St. to University Blvd."}

	t.Run("mapped", func(t *testing.T) {
		got, err := (&Locator{Geocoder: fakeGeocoder{}}).Locate(ctx, corridor)
		if err != nil || got.Status != project.Mapped || got.Position == nil ||
			got.Location == nil || got.Location.Kind != project.KindCorridor {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("no location in the text", func(t *testing.T) {
		p := project.Project{City: "denver", Title: "Denver Moves Everyone 2050", Description: "A citywide plan."}
		got, err := (&Locator{Geocoder: fakeGeocoder{}}).Locate(ctx, p)
		if err != nil || got.Status != project.NoLocationFound {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("a query with no match", func(t *testing.T) {
		g := fakeGeocoder{"Downing St & 1st Avenue": ErrNotFound}
		got, err := (&Locator{Geocoder: g}).Locate(ctx, corridor)
		if err != nil || got.Status != project.GeocodeFailed {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("a transient failure is an error, not a placement", func(t *testing.T) {
		boom := errors.New("quota")
		g := fakeGeocoder{"Downing St & 1st Avenue": boom}
		if _, err := (&Locator{Geocoder: g}).Locate(ctx, corridor); !errors.Is(err, boom) {
			t.Errorf("err = %v, want %v", err, boom)
		}
	})
}

func TestLocateUsesTheProjectsCity(t *testing.T) {
	ctx := context.Background()
	p := project.Project{City: "denver", Title: "1st Avenue Redesign", Description: "This project will redesign 1st Avenue from Downing St. to University Blvd."}

	t.Run("the geocoder is asked about the project's city", func(t *testing.T) {
		var asked []city.City
		g := geocoderFunc(func(_ string, in city.City) (project.Point, error) {
			asked = append(asked, in)
			return project.Point{Lat: 39.7392, Lng: -104.9903}, nil
		})
		got, err := (&Locator{Geocoder: g}).Locate(ctx, p)
		if err != nil || got.Status != project.Mapped || len(asked) != 2 || asked[0].ID != "denver" {
			t.Errorf("got %+v, %v, asked %v", got, err, asked)
		}
	})

	t.Run("a point outside the city is not mapped", func(t *testing.T) {
		golden := geocoderFunc(func(string, city.City) (project.Point, error) {
			return project.Point{Lat: 39.7294, Lng: -105.1319}, nil
		})
		got, err := (&Locator{Geocoder: golden}).Locate(ctx, p)
		if err != nil || got.Status != project.OutsideCity {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("an unknown city is an error", func(t *testing.T) {
		if _, err := (&Locator{Geocoder: fakeGeocoder{}}).Locate(ctx, project.Project{City: "nowhere", Title: "x"}); err == nil {
			t.Error("expected an error")
		}
	})
}

type geocoderFunc func(query string, in city.City) (project.Point, error)

func (f geocoderFunc) Geocode(_ context.Context, query string, in city.City) (project.Point, error) {
	return f(query, in)
}
