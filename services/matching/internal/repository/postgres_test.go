package repository

import (
	"context"
	"errors"
	"testing"

	"haillion/services/matching/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresLocationRepository_UpdateLocation(t *testing.T) {
	t.Run("successful upsert", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		driverID := uuid.New().String()
		loc := domain.Location{Lat: 37.7749, Lng: -122.4194}

		// ST_MakePoint takes (longitude, latitude) -> Lng then Lat
		mock.ExpectExec("INSERT INTO driver_locations").
			WithArgs(driverID, loc.Lng, loc.Lat).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.UpdateLocation(context.Background(), driverID, loc)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("upsert error returned", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		driverID := uuid.New().String()
		loc := domain.Location{Lat: 37.7749, Lng: -122.4194}

		dbErr := errors.New("write timeout")
		mock.ExpectExec("INSERT INTO driver_locations").
			WithArgs(driverID, loc.Lng, loc.Lat).
			WillReturnError(dbErr)

		err = repo.UpdateLocation(context.Background(), driverID, loc)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, dbErr))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresLocationRepository_FindNearbyDrivers(t *testing.T) {
	t.Run("successful query returning drivers", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		riderLoc := domain.Location{Lat: 37.7749, Lng: -122.4194}
		radius := 5000.0

		driverID1 := uuid.New().String()
		driverID2 := uuid.New().String()

		rows := sqlmock.NewRows([]string{"driver_id", "lat", "lng"}).
			AddRow(driverID1, 37.7750, -122.4190).
			AddRow(driverID2, 37.7760, -122.4200)

		mock.ExpectQuery("SELECT driver_id, ST_Y").
			WithArgs(riderLoc.Lng, riderLoc.Lat, radius).
			WillReturnRows(rows)

		drivers, err := repo.FindNearbyDrivers(context.Background(), riderLoc, radius)
		assert.NoError(t, err)
		require.Len(t, drivers, 2)
		assert.Equal(t, driverID1, drivers[0].DriverID)
		assert.Equal(t, 37.7750, drivers[0].Location.Lat)
		assert.Equal(t, -122.4190, drivers[0].Location.Lng)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice on no matches", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		riderLoc := domain.Location{Lat: 37.7749, Lng: -122.4194}
		radius := 5000.0

		// Empty rows
		rows := sqlmock.NewRows([]string{"driver_id", "lat", "lng"})

		mock.ExpectQuery("SELECT driver_id, ST_Y").
			WithArgs(riderLoc.Lng, riderLoc.Lat, radius).
			WillReturnRows(rows)

		drivers, err := repo.FindNearbyDrivers(context.Background(), riderLoc, radius)
		assert.NoError(t, err)
		assert.NotNil(t, drivers)
		assert.Len(t, drivers, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error returned", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		riderLoc := domain.Location{Lat: 37.7749, Lng: -122.4194}
		radius := 5000.0
		dbErr := errors.New("connection reset")

		mock.ExpectQuery("SELECT driver_id, ST_Y").
			WithArgs(riderLoc.Lng, riderLoc.Lat, radius).
			WillReturnError(dbErr)

		drivers, err := repo.FindNearbyDrivers(context.Background(), riderLoc, radius)
		assert.Error(t, err)
		assert.Nil(t, drivers)
		assert.True(t, errors.Is(err, dbErr))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
