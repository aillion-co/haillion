package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

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

func TestPostgresLocationRepository_UpsertPendingRider(t *testing.T) {
	t.Run("successful upsert pending rider", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		riderID := "rider-123"
		loc := domain.Location{Lat: 51.5074, Lng: -0.1278}

		mock.ExpectExec("INSERT INTO pending_riders").
			WithArgs(riderID, loc.Lng, loc.Lat).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.UpsertPendingRider(context.Background(), riderID, loc)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresLocationRepository_DeletePendingRider(t *testing.T) {
	t.Run("successful delete pending rider", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		riderID := "rider-123"

		mock.ExpectExec("DELETE FROM pending_riders").
			WithArgs(riderID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.DeletePendingRider(context.Background(), riderID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresLocationRepository_GetPendingRider(t *testing.T) {
	t.Run("get pending rider found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		riderID := "rider-123"

		rows := sqlmock.NewRows([]string{"lat", "lng"}).AddRow(51.5074, -0.1278)
		mock.ExpectQuery("SELECT ST_Y").
			WithArgs(riderID).
			WillReturnRows(rows)

		loc, err := repo.GetPendingRider(context.Background(), riderID)
		assert.NoError(t, err)
		assert.Equal(t, 51.5074, loc.Lat)
		assert.Equal(t, -0.1278, loc.Lng)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get pending rider not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		riderID := "rider-123"

		mock.ExpectQuery("SELECT ST_Y").
			WithArgs(riderID).
			WillReturnError(sql.ErrNoRows)

		_, err = repo.GetPendingRider(context.Background(), riderID)
		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresLocationRepository_GetDriverLocation(t *testing.T) {
	t.Run("get driver location found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		driverID := "driver-123"

		rows := sqlmock.NewRows([]string{"lat", "lng"}).AddRow(51.5074, -0.1278)
		mock.ExpectQuery("SELECT ST_Y").
			WithArgs(driverID).
			WillReturnRows(rows)

		loc, err := repo.GetDriverLocation(context.Background(), driverID)
		assert.NoError(t, err)
		assert.Equal(t, 51.5074, loc.Lat)
		assert.Equal(t, -0.1278, loc.Lng)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get driver location not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		driverID := "driver-123"

		mock.ExpectQuery("SELECT ST_Y").
			WithArgs(driverID).
			WillReturnError(sql.ErrNoRows)

		_, err = repo.GetDriverLocation(context.Background(), driverID)
		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresLocationRepository_FindNearbyPendingRiders(t *testing.T) {
	t.Run("successful query returning riders", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		center := domain.Location{Lat: 51.5074, Lng: -0.1278}
		radius := 5000.0
		maxAge := 10 * time.Minute

		rows := sqlmock.NewRows([]string{"rider_id", "lat", "lng", "distance_m"}).
			AddRow("rider-1", 51.5080, -0.1270, 150.0).
			AddRow("rider-2", 51.5090, -0.1260, 300.0)

		mock.ExpectQuery("SELECT rider_id, ST_Y").
			WithArgs(center.Lng, center.Lat, radius, maxAge.Seconds()).
			WillReturnRows(rows)

		hits, err := repo.FindNearbyPendingRiders(context.Background(), center, radius, maxAge)
		assert.NoError(t, err)
		require.Len(t, hits, 2)
		assert.Equal(t, "rider-1", hits[0].RiderID)
		assert.Equal(t, 150.0, hits[0].DistanceMeters)
		assert.Equal(t, "rider-2", hits[1].RiderID)
		assert.Equal(t, 300.0, hits[1].DistanceMeters)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresLocationRepository_FindNearbyDriversWithDistance(t *testing.T) {
	t.Run("successful query returning drivers with distance", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresLocationRepository(db)
		center := domain.Location{Lat: 51.5074, Lng: -0.1278}
		radius := 5000.0

		rows := sqlmock.NewRows([]string{"driver_id", "lat", "lng", "distance_m"}).
			AddRow("driver-1", 51.5080, -0.1270, 150.0).
			AddRow("driver-2", 51.5090, -0.1260, 300.0)

		mock.ExpectQuery("SELECT driver_id, ST_Y").
			WithArgs(center.Lng, center.Lat, radius).
			WillReturnRows(rows)

		hits, err := repo.FindNearbyDriversWithDistance(context.Background(), center, radius)
		assert.NoError(t, err)
		require.Len(t, hits, 2)
		assert.Equal(t, "driver-1", hits[0].DriverID)
		assert.Equal(t, 150.0, hits[0].DistanceMeters)
		assert.Equal(t, "driver-2", hits[1].DriverID)
		assert.Equal(t, 300.0, hits[1].DistanceMeters)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
