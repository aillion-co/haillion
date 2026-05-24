CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL,
    reviewer_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    score INT NOT NULL CHECK (score >= 1 AND score <= 5),
    comment TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT idx_reviews_trip_reviewer UNIQUE (trip_id, reviewer_id)
);
