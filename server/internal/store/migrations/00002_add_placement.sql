-- +goose Up
-- Where each project goes on the map, filled in by the refresher after a scrape.
-- Existing rows start out pending and are geocoded on the next refresh.
ALTER TABLE projects
    ADD COLUMN geocode_status text NOT NULL DEFAULT 'pending'
        CHECK (geocode_status IN ('pending', 'mapped', 'no-location-found', 'geocode-failed', 'outside-denver')),
    ADD COLUMN location_kind  text,
    ADD COLUMN location_label text,
    ADD COLUMN lat            double precision,
    ADD COLUMN lng            double precision;

-- +goose Down
ALTER TABLE projects
    DROP COLUMN geocode_status,
    DROP COLUMN location_kind,
    DROP COLUMN location_label,
    DROP COLUMN lat,
    DROP COLUMN lng;
