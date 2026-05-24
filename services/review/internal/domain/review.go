package domain

import (
	"context"
	"errors"
)

// ErrDuplicateReview is returned when a reviewer attempts to review the same trip twice.
var ErrDuplicateReview = errors.New("review already exists for this trip")

// ErrInvalidScore is returned when a score is outside the 1-5 stars range.
var ErrInvalidScore = errors.New("score must be between 1 and 5")

type Review struct {
	ID         string
	TripID     string
	ReviewerID string // Who wrote it
	SubjectID  string // Who it is about
	Score      int    // 1 to 5
	Comment    string
}

type ReviewRepository interface {
	Create(ctx context.Context, r *Review) error
	GetAverageScore(ctx context.Context, subjectID string) (float64, error)
}
