package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"haillion/services/review/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresReviewRepository struct {
	db *sql.DB
}

func NewPostgresReviewRepository(db *sql.DB) *PostgresReviewRepository {
	return &PostgresReviewRepository{db: db}
}

func (r *PostgresReviewRepository) Create(ctx context.Context, rev *domain.Review) error {
	if rev.Score < 1 || rev.Score > 5 {
		return fmt.Errorf("create review: %w", domain.ErrInvalidScore)
	}

	query := `
		INSERT INTO reviews (id, trip_id, reviewer_id, subject_id, score, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, rev.ID, rev.TripID, rev.ReviewerID, rev.SubjectID, rev.Score, rev.Comment)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation in Postgres
				return fmt.Errorf("create review: %w", domain.ErrDuplicateReview)
			}
		}
		return fmt.Errorf("insert review: %w", err)
	}
	return nil
}

func (r *PostgresReviewRepository) GetAverageScore(ctx context.Context, subjectID string) (float64, error) {
	query := `SELECT COALESCE(AVG(score), 0.0) FROM reviews WHERE subject_id = $1`
	var avg float64
	err := r.db.QueryRowContext(ctx, query, subjectID).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("calculate average score: %w", err)
	}
	return avg, nil
}

// Ensure PostgresReviewRepository implements domain.ReviewRepository
var _ domain.ReviewRepository = (*PostgresReviewRepository)(nil)
