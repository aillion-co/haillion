package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"aillion/services/identity/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (id, email, role) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, u.ID, u.Email, string(u.Role))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation in Postgres
				return fmt.Errorf("create user: %w", domain.ErrDuplicateEmail)
			}
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, role FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.User
	var roleStr string
	err := row.Scan(&u.ID, &u.Email, &roleStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get user: %w", domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = domain.Role(roleStr)
	return &u, nil
}
