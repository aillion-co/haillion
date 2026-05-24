package repository

import (
	"context"
	"errors"
	"testing"

	"haillion/services/review/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresReviewRepository_Create(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresReviewRepository(db)
		r := &domain.Review{
			ID:         uuid.New().String(),
			TripID:     uuid.New().String(),
			ReviewerID: uuid.New().String(),
			SubjectID:  uuid.New().String(),
			Score:      5,
			Comment:    "Fantastic driver!",
		}

		mock.ExpectExec("INSERT INTO reviews").
			WithArgs(r.ID, r.TripID, r.ReviewerID, r.SubjectID, r.Score, r.Comment).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(context.Background(), r)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid score constraints", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresReviewRepository(db)
		r := &domain.Review{
			ID:         uuid.New().String(),
			TripID:     uuid.New().String(),
			ReviewerID: uuid.New().String(),
			SubjectID:  uuid.New().String(),
			Score:      6, // Invalid
			Comment:    "Invalid",
		}

		err = repo.Create(context.Background(), r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInvalidScore))
	})

	t.Run("duplicate review conflict", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresReviewRepository(db)
		r := &domain.Review{
			ID:         uuid.New().String(),
			TripID:     uuid.New().String(),
			ReviewerID: uuid.New().String(),
			SubjectID:  uuid.New().String(),
			Score:      4,
			Comment:    "Dup",
		}

		pgErr := &pgconn.PgError{
			Code: "23505",
		}

		mock.ExpectExec("INSERT INTO reviews").
			WithArgs(r.ID, r.TripID, r.ReviewerID, r.SubjectID, r.Score, r.Comment).
			WillReturnError(pgErr)

		err = repo.Create(context.Background(), r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrDuplicateReview))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresReviewRepository_GetAverageScore(t *testing.T) {
	t.Run("successful calculation", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresReviewRepository(db)
		subjectID := uuid.New().String()

		rows := sqlmock.NewRows([]string{"avg"}).AddRow(4.5)
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(subjectID).
			WillReturnRows(rows)

		avg, err := repo.GetAverageScore(context.Background(), subjectID)
		assert.NoError(t, err)
		assert.InDelta(t, 4.5, avg, 0.001)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty matches calculates 0.0", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresReviewRepository(db)
		subjectID := uuid.New().String()

		rows := sqlmock.NewRows([]string{"avg"}).AddRow(0.0)
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(subjectID).
			WillReturnRows(rows)

		avg, err := repo.GetAverageScore(context.Background(), subjectID)
		assert.NoError(t, err)
		assert.InDelta(t, 0.0, avg, 0.001)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
