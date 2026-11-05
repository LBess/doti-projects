// Package geocode turns a project's text into a map position: it guesses a
// location (see package location), geocodes it with Google, and checks that
// the result lands inside the city.
package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/LBess/doti-projects/server/internal/city"
	"github.com/LBess/doti-projects/server/internal/project"
)

const defaultBaseURL = "https://maps.googleapis.com/maps/api/geocode/json"

// ErrNotFound means Google understood the request but has no match for the
// query. Unlike every other error from Geocode it is a permanent answer, not
// a reason to retry later.
var ErrNotFound = errors.New("no geocoding results")

// Client calls the Google Maps Geocoding API.
type Client struct {
	APIKey  string
	HTTP    *http.Client
	BaseURL string // overridden in tests
}

func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey, HTTP: &http.Client{Timeout: 10 * time.Second}, BaseURL: defaultBaseURL}
}

type response struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	Results      []struct {
		Geometry struct {
			Location project.Point `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
}

// Geocode resolves query to a coordinate, biased toward the city (which also
// keeps ambiguous street names like "Broadway" from resolving elsewhere). It
// returns ErrNotFound when there is no match; any other error (network
// trouble, quota, a rejected key) is worth retrying on a later run.
func (c *Client) Geocode(ctx context.Context, query string, in city.City) (project.Point, error) {
	params := url.Values{
		"address": {query + ", " + in.Label},
		"bounds":  {in.GeocodeBounds},
		"region":  {"us"},
		"key":     {c.APIKey},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"?"+params.Encode(), nil)
	if err != nil {
		return project.Point{}, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return project.Point{}, fmt.Errorf("geocode %q: %w", query, redactKey(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return project.Point{}, fmt.Errorf("geocode %q: unexpected status %s", query, resp.Status)
	}

	var body response
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return project.Point{}, fmt.Errorf("geocode %q: decode response: %w", query, err)
	}
	switch body.Status {
	case "OK":
		if len(body.Results) == 0 {
			return project.Point{}, ErrNotFound
		}
		return body.Results[0].Geometry.Location, nil
	case "ZERO_RESULTS":
		return project.Point{}, ErrNotFound
	default:
		return project.Point{}, fmt.Errorf("geocode %q: %s: %s", query, body.Status, body.ErrorMessage)
	}
}

// redactKey strips the request URL from a transport error, since net/http
// includes it and it carries the API key.
func redactKey(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
