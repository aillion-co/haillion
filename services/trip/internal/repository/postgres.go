package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"haillion/services/trip/internal/domain"
)

type PostgresTripRepository struct {
	db *sql.DB
}

func NewPostgresTripRepository(db *sql.DB) *PostgresTripRepository {
	return &PostgresTripRepository{db: db}
}

func (r *PostgresTripRepository) Create(ctx context.Context, t *domain.Trip) error {
	query := `
		INSERT INTO trips (id, rider_id, driver_id, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := r.db.ExecContext(ctx, query, t.ID, t.RiderID, t.DriverID, string(domain.StateRequested))
	if err != nil {
		return fmt.Errorf("create trip: %w", err)
	}
	t.State = domain.StateRequested
	return nil
}

func (r *PostgresTripRepository) UpdateState(ctx context.Context, id string, driverID *string, newState domain.State, expectedCurrent domain.State) error {
	query := `
		UPDATE trips
		SET state = $1, driver_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND state = $4
	`
	res, err := r.db.ExecContext(ctx, query, string(newState), driverID, id, string(expectedCurrent))
	if err != nil {
		return fmt.Errorf("update trip state: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		// Verify if the trip exists to return correct error
		var exists bool
		err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM trips WHERE id = $1)", id).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check trip existence: %w", err)
		}

		if !exists {
			return fmt.Errorf("update trip: %w", domain.ErrTripNotFound)
		}

		return fmt.Errorf("update trip: %w", domain.ErrInvalidStateTransition)
	}

	return nil
}

func (r *PostgresTripRepository) GetByID(ctx context.Context, id string) (*domain.Trip, error) {
	query := `SELECT id, rider_id, driver_id, state FROM trips WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var t domain.Trip
	var driverID sql.NullString
	var stateStr string

	err := row.Scan(&t.ID, &t.RiderID, &driverID, &stateStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get trip: %w", domain.ErrTripNotFound)
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}

	if driverID.Valid {
		s := driverID.String
		t.DriverID = &s
	}
	t.State = domain.State(stateStr)

	return &t, nil
}

// Ensure PostgresTripRepository implements domain.TripRepository
var _ domain.TripRepository = (*PostgresTripRepository)(nil)
