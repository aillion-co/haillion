package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"haillion/services/matching/internal/domain"
)

type PostgresLocationRepository struct {
	db *sql.DB
}

func NewPostgresLocationRepository(db *sql.DB) *PostgresLocationRepository {
	return &PostgresLocationRepository{db: db}
}

func (r *PostgresLocationRepository) UpdateLocation(ctx context.Context, driverID string, loc domain.Location) error {
	// PostGIS ST_MakePoint takes (longitude, latitude)
	query := `
		INSERT INTO driver_locations (driver_id, geom, updated_at)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, CURRENT_TIMESTAMP)
		ON CONFLICT (driver_id)
		DO UPDATE SET geom = EXCLUDED.geom, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, driverID, loc.Lng, loc.Lat)
	if err != nil {
		return fmt.Errorf("upsert driver location: %w", err)
	}
	return nil
}

func (r *PostgresLocationRepository) FindNearbyDrivers(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
	// Query uses ST_DWithin and orders by ST_Distance
	// ST_MakePoint takes (longitude, latitude)
	query := `
		SELECT driver_id, ST_Y(geom::geometry) AS lat, ST_X(geom::geometry) AS lng
		FROM driver_locations
		WHERE ST_DWithin(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY ST_Distance(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) ASC
	`
	rows, err := r.db.QueryContext(ctx, query, riderLoc.Lng, riderLoc.Lat, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("query nearby drivers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var drivers []domain.DriverLocation
	for rows.Next() {
		var dl domain.DriverLocation
		err := rows.Scan(&dl.DriverID, &dl.Location.Lat, &dl.Location.Lng)
		if err != nil {
			return nil, fmt.Errorf("scan driver location: %w", err)
		}
		drivers = append(drivers, dl)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	// Always return empty slice instead of nil if no drivers are found, per AC
	if drivers == nil {
		return []domain.DriverLocation{}, nil
	}

	return drivers, nil
}

func (r *PostgresLocationRepository) UpsertPendingRider(ctx context.Context, riderID string, loc domain.Location) error {
	query := `
		INSERT INTO pending_riders (rider_id, geom, updated_at)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, CURRENT_TIMESTAMP)
		ON CONFLICT (rider_id)
		DO UPDATE SET geom = EXCLUDED.geom, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, riderID, loc.Lng, loc.Lat)
	if err != nil {
		return fmt.Errorf("upsert pending rider: %w", err)
	}
	return nil
}

func (r *PostgresLocationRepository) DeletePendingRider(ctx context.Context, riderID string) error {
	query := `DELETE FROM pending_riders WHERE rider_id = $1`
	_, err := r.db.ExecContext(ctx, query, riderID)
	if err != nil {
		return fmt.Errorf("delete pending rider: %w", err)
	}
	return nil
}

func (r *PostgresLocationRepository) GetPendingRider(ctx context.Context, riderID string) (domain.Location, error) {
	query := `
		SELECT ST_Y(geom::geometry) AS lat, ST_X(geom::geometry) AS lng
		FROM pending_riders
		WHERE rider_id = $1
	`
	var loc domain.Location
	err := r.db.QueryRowContext(ctx, query, riderID).Scan(&loc.Lat, &loc.Lng)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Location{}, domain.ErrNotFound
		}
		return domain.Location{}, fmt.Errorf("get pending rider: %w", err)
	}
	return loc, nil
}

func (r *PostgresLocationRepository) GetDriverLocation(ctx context.Context, driverID string) (domain.Location, error) {
	query := `
		SELECT ST_Y(geom::geometry) AS lat, ST_X(geom::geometry) AS lng
		FROM driver_locations
		WHERE driver_id = $1
	`
	var loc domain.Location
	err := r.db.QueryRowContext(ctx, query, driverID).Scan(&loc.Lat, &loc.Lng)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Location{}, domain.ErrNotFound
		}
		return domain.Location{}, fmt.Errorf("get driver location: %w", err)
	}
	return loc, nil
}

func (r *PostgresLocationRepository) FindNearbyPendingRiders(ctx context.Context, center domain.Location, radiusMeters float64, maxAge time.Duration) ([]domain.PendingRiderHit, error) {
	query := `
		SELECT rider_id, ST_Y(geom::geometry) AS lat, ST_X(geom::geometry) AS lng,
		       ST_Distance(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance_m
		FROM pending_riders
		WHERE ST_DWithin(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		  AND NOW() - updated_at < $4 * INTERVAL '1 second'
		ORDER BY distance_m ASC
	`
	rows, err := r.db.QueryContext(ctx, query, center.Lng, center.Lat, radiusMeters, maxAge.Seconds())
	if err != nil {
		return nil, fmt.Errorf("query nearby pending riders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var hits []domain.PendingRiderHit
	for rows.Next() {
		var h domain.PendingRiderHit
		err := rows.Scan(&h.RiderID, &h.Location.Lat, &h.Location.Lng, &h.DistanceMeters)
		if err != nil {
			return nil, fmt.Errorf("scan pending rider hit: %w", err)
		}
		hits = append(hits, h)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if hits == nil {
		return []domain.PendingRiderHit{}, nil
	}
	return hits, nil
}

func (r *PostgresLocationRepository) FindNearbyDriversWithDistance(ctx context.Context, center domain.Location, radiusMeters float64) ([]domain.DriverLocationHit, error) {
	query := `
		SELECT driver_id, ST_Y(geom::geometry) AS lat, ST_X(geom::geometry) AS lng,
		       ST_Distance(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance_m
		FROM driver_locations
		WHERE ST_DWithin(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY distance_m ASC
	`
	rows, err := r.db.QueryContext(ctx, query, center.Lng, center.Lat, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("query nearby drivers with distance: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var hits []domain.DriverLocationHit
	for rows.Next() {
		var h domain.DriverLocationHit
		err := rows.Scan(&h.DriverID, &h.Location.Lat, &h.Location.Lng, &h.DistanceMeters)
		if err != nil {
			return nil, fmt.Errorf("scan driver location hit: %w", err)
		}
		hits = append(hits, h)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if hits == nil {
		return []domain.DriverLocationHit{}, nil
	}
	return hits, nil
}

// Ensure PostgresLocationRepository implements domain.LocationRepository
var _ domain.LocationRepository = (*PostgresLocationRepository)(nil)
