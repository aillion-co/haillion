package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"haillion/services/trip/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresTripRepository_Create(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresTripRepository(db)
		tr := &domain.Trip{
			ID:      uuid.New().String(),
			RiderID: uuid.New().String(),
		}

		mock.ExpectExec("INSERT INTO trips").
			WithArgs(tr.ID, tr.RiderID, tr.DriverID, string(domain.StateRequested)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(context.Background(), tr)
		assert.NoError(t, err)
		assert.Equal(t, domain.StateRequested, tr.State)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresTripRepository_UpdateState(t *testing.T) {
	t.Run("successful state transition", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresTripRepository(db)
		tripID := uuid.New().String()
		driverID := uuid.New().String()

		mock.ExpectExec("UPDATE trips").
			WithArgs(string(domain.StateAccepted), &driverID, tripID, string(domain.StateRequested)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.UpdateState(context.Background(), tripID, &driverID, domain.StateAccepted, domain.StateRequested)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid state transition (concurrency collision)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresTripRepository(db)
		tripID := uuid.New().String()
		driverID := uuid.New().String()

		// RowsAffected = 0
		mock.ExpectExec("UPDATE trips").
			WithArgs(string(domain.StateAccepted), &driverID, tripID, string(domain.StateRequested)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Check existence check query -> returns true (trip exists)
		rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(tripID).
			WillReturnRows(rows)

		err = repo.UpdateState(context.Background(), tripID, &driverID, domain.StateAccepted, domain.StateRequested)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInvalidStateTransition))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("transition fails because trip not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresTripRepository(db)
		tripID := uuid.New().String()
		driverID := uuid.New().String()

		// RowsAffected = 0
		mock.ExpectExec("UPDATE trips").
			WithArgs(string(domain.StateAccepted), &driverID, tripID, string(domain.StateRequested)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Check existence check query -> returns false (trip does not exist)
		rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(tripID).
			WillReturnRows(rows)

		err = repo.UpdateState(context.Background(), tripID, &driverID, domain.StateAccepted, domain.StateRequested)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrTripNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresTripRepository_GetByID(t *testing.T) {
	t.Run("successful retrieval with driver", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresTripRepository(db)
		tripID := uuid.New().String()
		riderID := uuid.New().String()
		driverID := uuid.New().String()

		rows := sqlmock.NewRows([]string{"id", "rider_id", "driver_id", "state"}).
			AddRow(tripID, riderID, driverID, string(domain.StateAccepted))

		mock.ExpectQuery("SELECT id, rider_id, driver_id, state FROM trips").
			WithArgs(tripID).
			WillReturnRows(rows)

		tr, err := repo.GetByID(context.Background(), tripID)
		assert.NoError(t, err)
		require.NotNil(t, tr)
		assert.Equal(t, tripID, tr.ID)
		assert.Equal(t, riderID, tr.RiderID)
		require.NotNil(t, tr.DriverID)
		assert.Equal(t, driverID, *tr.DriverID)
		assert.Equal(t, domain.StateAccepted, tr.State)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("trip not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresTripRepository(db)
		tripID := uuid.New().String()

		mock.ExpectQuery("SELECT id, rider_id, driver_id, state FROM trips").
			WithArgs(tripID).
			WillReturnError(sql.ErrNoRows)

		tr, err := repo.GetByID(context.Background(), tripID)
		assert.Error(t, err)
		assert.Nil(t, tr)
		assert.True(t, errors.Is(err, domain.ErrTripNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
