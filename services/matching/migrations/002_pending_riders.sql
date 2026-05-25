CREATE TABLE pending_riders (
    rider_id   TEXT PRIMARY KEY,
    geom       GEOGRAPHY(POINT, 4326) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX pending_riders_geom_idx ON pending_riders USING GIST (geom);
