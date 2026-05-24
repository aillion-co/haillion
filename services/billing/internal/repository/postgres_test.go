package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"haillion/services/billing/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresPaymentRepository_Create(t *testing.T) {
	t.Run("successful payment creation as pending", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresPaymentRepository(db)
		p := &domain.Payment{
			ID:     uuid.New().String(),
			TripID: uuid.New().String(),
			Amount: 2500, // $25.00
		}

		mock.ExpectExec("INSERT INTO payments").
			WithArgs(p.ID, p.TripID, p.Amount, string(domain.PaymentStatePending)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(context.Background(), p)
		assert.NoError(t, err)
		assert.Equal(t, domain.PaymentStatePending, p.State)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresPaymentRepository_MarkPaid(t *testing.T) {
	t.Run("successful transition to paid", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresPaymentRepository(db)
		paymentID := uuid.New().String()

		mock.ExpectExec("UPDATE payments").
			WithArgs(string(domain.PaymentStatePaid), paymentID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.MarkPaid(context.Background(), paymentID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("payment not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresPaymentRepository(db)
		paymentID := uuid.New().String()

		mock.ExpectExec("UPDATE payments").
			WithArgs(string(domain.PaymentStatePaid), paymentID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = repo.MarkPaid(context.Background(), paymentID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrPaymentNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresPaymentRepository_GetByID(t *testing.T) {
	t.Run("successful retrieve", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresPaymentRepository(db)
		paymentID := uuid.New().String()
		tripID := uuid.New().String()

		rows := sqlmock.NewRows([]string{"id", "trip_id", "amount", "state"}).
			AddRow(paymentID, tripID, int64(3000), string(domain.PaymentStatePaid))

		mock.ExpectQuery("SELECT id, trip_id, amount, state FROM payments").
			WithArgs(paymentID).
			WillReturnRows(rows)

		p, err := repo.GetByID(context.Background(), paymentID)
		assert.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, paymentID, p.ID)
		assert.Equal(t, tripID, p.TripID)
		assert.Equal(t, int64(3000), p.Amount)
		assert.Equal(t, domain.PaymentStatePaid, p.State)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		repo := NewPostgresPaymentRepository(db)
		paymentID := uuid.New().String()

		mock.ExpectQuery("SELECT id, trip_id, amount, state FROM payments").
			WithArgs(paymentID).
			WillReturnError(sql.ErrNoRows)

		p, err := repo.GetByID(context.Background(), paymentID)
		assert.Error(t, err)
		assert.Nil(t, p)
		assert.True(t, errors.Is(err, domain.ErrPaymentNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCalculateSurge(t *testing.T) {
	tests := []struct {
		name     string
		riders   int
		drivers  int
		expected float64
	}{
		{
			name:     "no drivers with waiting riders (high surge)",
			riders:   5,
			drivers:  0,
			expected: 2.0,
		},
		{
			name:     "no drivers and no riders",
			riders:   0,
			drivers:  0,
			expected: 1.0,
		},
		{
			name:     "more drivers than riders (equal/excess supply)",
			riders:   2,
			drivers:  4,
			expected: 1.0,
		},
		{
			name:     "equal riders and drivers",
			riders:   3,
			drivers:  3,
			expected: 1.0,
		},
		{
			name:     "high demand (ratio > 1)",
			riders:   10,
			drivers:  5,   // ratio = 2.0
			expected: 1.2, // 1.0 + (2.0 - 1.0)*0.2 = 1.2
		},
		{
			name:     "extreme demand capped at maximum 3.0",
			riders:   200,
			drivers:  10, // ratio = 20.0, multiplier = 1.0 + 19.0*0.2 = 4.8
			expected: 3.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := domain.CalculateSurge(tc.riders, tc.drivers)
			assert.InDelta(t, tc.expected, res, 0.001)
		})
	}
}
