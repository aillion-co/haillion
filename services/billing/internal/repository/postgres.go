package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"haillion/services/billing/internal/domain"
)

type PostgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	query := `
		INSERT INTO payments (id, trip_id, amount, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.TripID, p.Amount, string(domain.PaymentStatePending))
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	p.State = domain.PaymentStatePending
	return nil
}

func (r *PostgresPaymentRepository) MarkPaid(ctx context.Context, id string) error {
	query := `
		UPDATE payments
		SET state = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	res, err := r.db.ExecContext(ctx, query, string(domain.PaymentStatePaid), id)
	if err != nil {
		return fmt.Errorf("mark paid: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected check: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("mark paid: %w", domain.ErrPaymentNotFound)
	}

	return nil
}

func (r *PostgresPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `SELECT id, trip_id, amount, state FROM payments WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var p domain.Payment
	var stateStr string

	err := row.Scan(&p.ID, &p.TripID, &p.Amount, &stateStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get payment: %w", domain.ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("scan payment: %w", err)
	}
	p.State = domain.PaymentState(stateStr)

	return &p, nil
}

// Ensure PostgresPaymentRepository implements domain.PaymentRepository
var _ domain.PaymentRepository = (*PostgresPaymentRepository)(nil)
