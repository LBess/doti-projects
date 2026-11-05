# Denver DOTI Current Projects

A prototype map of Denver's Department of Transportation & Infrastructure
(DOTI) current projects. It pulls the list of projects from the
[DOTI Projects page](https://www.denvergov.org/Government/Agencies-Departments-Offices/Agencies-Departments-Offices-Directory/Department-of-Transportation-and-Infrastructure/Programs-Services/Projects),
guesses a location for each project, and drops a pin on a Google Map for every
project it can confidently place.

A small Go backend (`server/`) scrapes the DOTI page on a schedule, works out
where each project goes on the map, and stores the result in Postgres; the React
frontend fetches the projects from `GET /api/projects` and renders the map.

## How it works

1. **Scrape & store** (`server/`) — on startup and then every
   `REFRESH_INTERVAL` (default 12h), the Go server fetches the DOTI page,
   parses the **Transportation/Mobility** section, and reconciles the
   `projects` table with it: new projects are inserted, projects that
   disappeared from the page are marked `completed`, and ones that reappear
   are marked `active` again. A scrape that returns 0 projects, or under 50%
   of the currently active ones, is treated as a broken page/parser and
   ignored. See [Backend](#backend).
2. **Locate** (`server/internal/location`) — right after a scrape, pure regex
   heuristics guess a geocodable location for each new project from its
   title + description:
   - `"<street> from <A> to <B>"` and similar bounded-corridor phrasing → pin
     at the midpoint between the two cross streets.
   - `"<A> and/& <B>"` or `"<A> (bridge) over <B>"` → pin at that
     intersection.
   - Named neighborhoods, when no street can be found → pin at their centroid.
   - Anything else (city-wide programs, plans, studies with no single site)
     → no location guessed, so no pin. This is expected and common — roughly
     a third of DOTI's current projects are single-site; the rest are
     citywide/multi-neighborhood programs.
3. **Geocode** (`server/internal/geocode`) — location guesses are resolved to
   lat/lng via the Google Geocoding API, biased to the project's city, and a
   result that falls outside the city limit is discarded. The outcome is
   stored on the project, so each project is geocoded once: again only if its
   title or description changes, or if the attempt failed for a transient
   reason (network, quota), in which case it stays `pending` and is retried on
   the next refresh. Needs `GEOCODING_API_KEY`.
4. **Fetch** (`src/lib/fetchDotiProjects.ts`) — the browser calls
   `GET /api/projects`, which returns the active projects with their
   position (or the reason they have none). It's same-origin (Vite proxies
   `/api` in dev; the Go server serves the built app in production), so no
   CORS proxy is involved.
5. **Render** (`src/components/ProjectMap.tsx`,
   `src/components/UnmappedProjects.tsx`) — mapped projects get a marker
   with an info window; everything else is listed below the map with a link
   to its detail page.

Because step 2 is a heuristic over free-form prose, pin placement is a
best-effort guess, not authoritative — always cross-check against the
"Read more" link before treating a pin's position as precise.

## Setup

```bash
npm install
cp .env.example .env
```

Edit `.env` and set:

- `VITE_GOOGLE_MAPS_API_KEY` — a Google Maps API key with the **Maps
  JavaScript API** enabled. It ships in the browser bundle.
- `POSTGRES_PASSWORD` — any password for the local Postgres container.
- `GEOCODING_API_KEY` — a separate key with only the **Geocoding API** enabled,
  used by the Go server. Keep it out of the frontend build.

Start the backend (Go server + Postgres), then the frontend:

```bash
docker compose up --build   # API on :8080, Postgres on 127.0.0.1:5432
npm run dev                 # Vite proxies /api to :8080
```

The first scrape runs as soon as the server starts; the page shows an error
until it finishes (a second or two).

## Scripts

- `npm run dev` — start the dev server
- `npm run build` — typecheck + production build
- `cd server && go test ./...` — backend tests
- `npm run lint` — oxlint

## Known limitations

- **Location heuristics are approximate.** They cover the common phrasings
  DOTI uses ("from X to Y", "X and Y", "X over Y") but won't catch every
  project, and an occasional pin may land in a slightly wrong spot (e.g. a
  master-plan project that happens to mention one specific intersection).
- **Geocoding quota.** A "demo"/free-tier Google key may have limited
  geocoding quota. Each project is geocoded once and the result is stored, so
  steady-state usage is only the handful of projects DOTI adds between
  refreshes. Wiping the database re-geocodes everything.
- **Neighborhood fallback is Denver-only.** Other cities have no neighborhood
  list yet, so their projects are only pinned from streets/intersections.

## Cities

The app covers several cities in the Denver metro. Each city is an entry in
`server/internal/city` (id, name, boundary color, the geocoder's bias bounds,
its boundary polygon) and has its own scraper and its own refresh loop, so one
city's broken page never affects another's. Every project row carries its
`city`, and a project's identity is `(city, source_key)`.

Boundaries live once, in `server/internal/city/boundaries/<id>.json` (a GeoJSON
MultiPolygon's coordinates: polygons of an outer ring plus holes). The server
uses them to discard geocoded points outside the city, and serves them from
`GET /api/cities` for the frontend to draw in each city's color.

| City     | Source                                                                                                                  |
| -------- | ----------------------------------------------------------------------------------------------------------------------- |
| Denver   | the DOTI Projects page (`scraper.Fetcher`)                                                                              |
| Lakewood | [Cone Zone](https://www.lakewoodco.gov/Local-Government/Departments/Public-Works/Cone-Zone) (`scraper.LakewoodFetcher`) |

**Adding a city:** add `boundaries/<id>.json` (e.g. from Census TIGERweb's
Incorporated Places layer) and a `city.City` entry in `city.go`/`All`, write a
scraper that returns `[]project.Project`, and register it in
`cmd/server/main.go`. No frontend or migration change is needed.

## Backend

`server/` is a Go service (stdlib `net/http`, `goquery`, `pgx`, `goose`).

| Variable           | Default                    | Purpose                                                   |
| ------------------ | -------------------------- | --------------------------------------------------------- |
| `DATABASE_URL`     | — (required)               | Postgres connection string                                |
| `PORT`             | `8080`                     | HTTP port                                                 |
| `REFRESH_INTERVAL` | `12h`                      | How often to re-scrape the DOTI page                      |
| `DOTI_URL`         | the DOTI Projects page     | Denver page to scrape                                     |
| `LAKEWOOD_URL`     | Lakewood's Cone Zone page  | Lakewood page to scrape                                   |
| `STATIC_DIR`       | — (unset)                  | If set, also serve the built frontend from this directory |
| `GEOCODING_API_KEY`| — (unset)                  | Google Geocoding API key; without it projects get no pins |

Endpoints: `GET /api/projects` (active projects of every city; `?status=completed` for
finished ones), `GET /api/cities` and `GET /healthz`. Each project carries its `city` id,
`position` and
`location` (`kind` + `label`) when it could be mapped, and `unmappedReason`
(`pending`, `no-location-found`, `geocode-failed` or `outside-city`) when not. Migrations in
`server/internal/store/migrations` run automatically on startup. There's no
authentication — it's meant to be reachable only over the VPN.

Each project is keyed by a `source_key` derived from its detail URL (or its
title if it has none) rather than its position on the page; see
`sourceKey` in `server/internal/scraper/scraper.go`. Only the
Transportation/Mobility section is kept for now.

The store test needs a throwaway Postgres and is skipped otherwise:

```bash
TEST_DATABASE_URL='postgres://doti:doti@localhost:5432/doti_test?sslmode=disable' \
    go test ./internal/store
```

**Deploying:** `docker build -t doti-projects --build-arg VITE_GOOGLE_MAPS_API_KEY=... .`
builds the default `app` stage, a single image that serves both the API and the
built frontend. Run it next to Postgres with `DATABASE_URL` and `GEOCODING_API_KEY` set.
