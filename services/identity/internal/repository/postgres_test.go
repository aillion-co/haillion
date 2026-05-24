package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"haillion/services/identity/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresUserRepository_Create(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresUserRepository(db)
		u := &domain.User{
			ID:    uuid.New().String(),
			Email: "test@example.com",
			Role:  domain.RoleRider,
		}

		mock.ExpectExec("INSERT INTO users (id, email, role) VALUES ($1, $2, $3)").
			WithArgs(u.ID, u.Email, string(u.Role)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(context.Background(), u)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate email returns typed conflict error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresUserRepository(db)
		u := &domain.User{
			ID:    uuid.New().String(),
			Email: "duplicate@example.com",
			Role:  domain.RoleDriver,
		}

		// Mock standard postgres duplicate key violation error code (23505)
		pgErr := &pgconn.PgError{
			Code: "23505",
		}

		mock.ExpectExec("INSERT INTO users (id, email, role) VALUES ($1, $2, $3)").
			WithArgs(u.ID, u.Email, string(u.Role)).
			WillReturnError(pgErr)

		err = repo.Create(context.Background(), u)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrDuplicateEmail))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other database errors are returned wrapped", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresUserRepository(db)
		u := &domain.User{
			ID:    uuid.New().String(),
			Email: "test@example.com",
			Role:  domain.RoleRider,
		}

		dbErr := errors.New("connection reset")
		mock.ExpectExec("INSERT INTO users (id, email, role) VALUES ($1, $2, $3)").
			WithArgs(u.ID, u.Email, string(u.Role)).
			WillReturnError(dbErr)

		err = repo.Create(context.Background(), u)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, dbErr))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresUserRepository_GetByID(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresUserRepository(db)
		userID := uuid.New().String()
		expectedUser := &domain.User{
			ID:    userID,
			Email: "driver@example.com",
			Role:  domain.RoleDriver,
		}

		rows := sqlmock.NewRows([]string{"id", "email", "role"}).
			AddRow(expectedUser.ID, expectedUser.Email, string(expectedUser.Role))

		mock.ExpectQuery("SELECT id, email, role FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnRows(rows)

		user, err := repo.GetByID(context.Background(), userID)
		assert.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Email, user.Email)
		assert.Equal(t, expectedUser.Role, user.Role)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found returns typed error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresUserRepository(db)
		userID := uuid.New().String()

		mock.ExpectQuery("SELECT id, email, role FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		user, err := repo.GetByID(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, domain.ErrUserNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other select errors are returned wrapped", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresUserRepository(db)
		userID := uuid.New().String()
		dbErr := errors.New("db disconnect")

		mock.ExpectQuery("SELECT id, email, role FROM users WHERE id = $1").
			WithArgs(userID).
			WillReturnError(dbErr)

		user, err := repo.GetByID(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, dbErr))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
