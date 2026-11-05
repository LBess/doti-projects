-- +goose Up
-- Projects now belong to a city. Everything stored so far came from Denver's
-- DOTI page. A project's identity is its source_key within a city, since keys
-- derived from titles could collide between cities.
ALTER TABLE projects ADD COLUMN city text NOT NULL DEFAULT 'denver';
ALTER TABLE projects ALTER COLUMN city DROP DEFAULT;
ALTER TABLE projects DROP CONSTRAINT projects_pkey;
ALTER TABLE projects ADD PRIMARY KEY (city, source_key);

-- 'outside-denver' becomes 'outside-city' now that Denver isn't the only city.
ALTER TABLE projects DROP CONSTRAINT projects_geocode_status_check;
UPDATE projects SET geocode_status = 'outside-city' WHERE geocode_status = 'outside-denver';
ALTER TABLE projects
    ADD CONSTRAINT projects_geocode_status_check
        CHECK (geocode_status IN ('pending', 'mapped', 'no-location-found', 'geocode-failed', 'outside-city'));

-- +goose Down
DELETE FROM projects WHERE city <> 'denver';
ALTER TABLE projects DROP CONSTRAINT projects_geocode_status_check;
UPDATE projects SET geocode_status = 'outside-denver' WHERE geocode_status = 'outside-city';
ALTER TABLE projects
    ADD CONSTRAINT projects_geocode_status_check
        CHECK (geocode_status IN ('pending', 'mapped', 'no-location-found', 'geocode-failed', 'outside-denver'));
ALTER TABLE projects DROP CONSTRAINT projects_pkey;
ALTER TABLE projects ADD PRIMARY KEY (source_key);
ALTER TABLE projects DROP COLUMN city;
