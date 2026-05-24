package repository

import (
	"context"
	"database/sql"
	"fmt"

	"aillion/services/matching/internal/domain"
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

// Ensure PostgresLocationRepository implements domain.LocationRepository
var _ domain.LocationRepository = (*PostgresLocationRepository)(nil)
