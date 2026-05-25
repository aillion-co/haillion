package e2e

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define request and response shapes matching downstream handlers for self-containment.

type RegisterRequest struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateLocationRequest struct {
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
	Postcode string   `json:"postcode,omitempty"`
}

type MatchRequest struct {
	RiderID  string   `json:"rider_id"`
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
	Postcode string   `json:"postcode,omitempty"`
}

type RiderRequest struct {
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
	Postcode string   `json:"postcode,omitempty"`
}

type NearbyRiderJSON struct {
	RiderID   string  `json:"rider_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM float64 `json:"distance_m"`
}

type NearbyRidersResponse struct {
	Riders []NearbyRiderJSON `json:"riders"`
}

type FareBreakdown struct {
	DistanceMiles   float64 `json:"distance_miles"`
	BasePence       int64   `json:"base_pence"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	TotalPence      int64   `json:"total_pence"`
	MinimumApplied  bool    `json:"minimum_applied"`
}

type MatchResponse struct {
	DriverID string `json:"driver_id"`
	ETA      int    `json:"eta_seconds"`
}

type CreateTripRequest struct {
	RiderID string  `json:"rider_id"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

type AcceptTripRequest struct {
	DriverID string `json:"driver_id"`
}

type TripResponse struct {
	ID       string  `json:"id"`
	RiderID  string  `json:"rider_id"`
	DriverID *string `json:"driver_id,omitempty"`
	State    string  `json:"state"`
}

type CreatePaymentRequest struct {
	TripID string `json:"trip_id"`
	Amount int64  `json:"amount"`
}

type PaymentResponse struct {
	ID     string `json:"id"`
	TripID string `json:"trip_id"`
	Amount int64  `json:"amount"`
	State  string `json:"state"`
}

type CreateReviewRequest struct {
	TripID    string `json:"trip_id"`
	SubjectID string `json:"subject_id"`
	Score     int    `json:"score"`
	Comment   string `json:"comment"`
}

type ReviewResponse struct {
	ID         string `json:"id"`
	TripID     string `json:"trip_id"`
	ReviewerID string `json:"reviewer_id"`
	SubjectID  string `json:"subject_id"`
	Score      int    `json:"score"`
	Comment    string `json:"comment"`
}

type RatingResponse struct {
	Average float64 `json:"average"`
}

func TestE2E_PlatformJourney(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	client := NewClient(gatewayURL)
	ctx := context.Background()

	// Generate unique IDs and emails to prevent database conflicts when tests run multiple times.
	riderID := uuid.New().String()
	riderEmail := fmt.Sprintf("rider-%s@example.com", riderID)

	driverID := uuid.New().String()
	driverEmail := fmt.Sprintf("driver-%s@example.com", driverID)

	var tripID string
	var paymentID string

	// Coordinates for matching (set to EC1A centroid since postcode matching places Rider at EC1A)
	testLat := 51.5202
	testLng := -0.104412

	// 1. Register Rider
	t.Run("RegisterRider", func(t *testing.T) {
		reqBody := RegisterRequest{
			ID:    riderID,
			Email: riderEmail,
			Role:  "rider",
		}
		var resp UserResponse
		code, err := client.Request(ctx, "POST", "/api/identity/users/register", nil, reqBody, &resp)
		require.NoError(t, err, "failed to register rider")
		assert.Equal(t, 201, code, "expected HTTP 201 Created for rider registration")
		assert.Equal(t, riderID, resp.ID)
		assert.Equal(t, riderEmail, resp.Email)
		assert.Equal(t, "rider", resp.Role)
	})

	// 2. Register Driver
	t.Run("RegisterDriver", func(t *testing.T) {
		reqBody := RegisterRequest{
			ID:    driverID,
			Email: driverEmail,
			Role:  "driver",
		}
		var resp UserResponse
		code, err := client.Request(ctx, "POST", "/api/identity/users/register", nil, reqBody, &resp)
		require.NoError(t, err, "failed to register driver")
		assert.Equal(t, 201, code, "expected HTTP 201 Created for driver registration")
		assert.Equal(t, driverID, resp.ID)
		assert.Equal(t, driverEmail, resp.Email)
		assert.Equal(t, "driver", resp.Role)
	})

	// 2b. Fare Estimate
	t.Run("FareEstimate", func(t *testing.T) {
		var fareResp FareBreakdown
		code, err := client.Request(ctx, "GET", "/api/billing/fare/estimate?from=SW1A&to=EC1A&riders=1&drivers=1", nil, nil, &fareResp)
		require.NoError(t, err, "failed to get fare estimate")
		assert.Equal(t, 200, code)

		// Asserts distance_miles is within 0.5 of ~2 mi (London Westminster ↔ Clerkenwell)
		assert.InDelta(t, 2.0, fareResp.DistanceMiles, 0.5)
		assert.Equal(t, 1.0, fareResp.SurgeMultiplier)

		expectedTotalPence := int64(math.Round(fareResp.DistanceMiles * 100))
		if expectedTotalPence < 250 {
			expectedTotalPence = 250
		}
		assert.Equal(t, expectedTotalPence, fareResp.TotalPence)

		// Call again with riders=10&drivers=2
		var fareResp2 FareBreakdown
		code2, err2 := client.Request(ctx, "GET", "/api/billing/fare/estimate?from=SW1A&to=EC1A&riders=10&drivers=2", nil, nil, &fareResp2)
		require.NoError(t, err2, "failed to get surged fare estimate")
		assert.Equal(t, 200, code2)

		assert.Greater(t, fareResp2.SurgeMultiplier, 1.0)
		assert.Greater(t, fareResp2.TotalPence, fareResp2.BasePence)
	})

	// 3. Driver Updates Location
	t.Run("DriverUpdatesLocation", func(t *testing.T) {
		reqBody := UpdateLocationRequest{
			Postcode: "SW1A",
		}
		code, err := client.Request(ctx, "POST", fmt.Sprintf("/api/matching/drivers/%s/location", driverID), nil, reqBody, nil)
		require.NoError(t, err, "failed to update driver location")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for driver location update")
	})

	// 3b. Nearby Riders
	t.Run("NearbyRiders", func(t *testing.T) {
		// 1. Post Rider Request at EC1A
		riderReq := RiderRequest{
			Postcode: "EC1A",
		}
		code, err := client.Request(ctx, "POST", fmt.Sprintf("/api/matching/riders/%s/request", riderID), nil, riderReq, nil)
		require.NoError(t, err, "failed to request rider match")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for rider match request")

		// 2. Query Nearby Riders for Driver
		var nearbyResp NearbyRidersResponse
		code, err = client.Request(ctx, "GET", fmt.Sprintf("/api/matching/drivers/%s/nearby-riders", driverID), nil, nil, &nearbyResp)
		require.NoError(t, err, "failed to get nearby riders")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for nearby riders query")

		// 3. Assert Rider Appears in the response
		found := false
		for _, r := range nearbyResp.Riders {
			if r.RiderID == riderID {
				found = true
				break
			}
		}
		assert.True(t, found, "expected rider %s to be in nearby riders list", riderID)

		// 4. Cancel Rider Request to clean up state
		code, err = client.Request(ctx, "DELETE", fmt.Sprintf("/api/matching/riders/%s/request", riderID), nil, nil, nil)
		require.NoError(t, err, "failed to cancel rider request")
		assert.Equal(t, 204, code, "expected HTTP 204 No Content for rider cancel")
	})

	// 4. Request Driver Match
	t.Run("RequestDriverMatch", func(t *testing.T) {
		reqBody := MatchRequest{
			RiderID:  riderID,
			Postcode: "EC1A",
		}
		var resp MatchResponse
		code, err := client.Request(ctx, "POST", "/api/matching/match", nil, reqBody, &resp)
		require.NoError(t, err, "failed to match driver")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for driver matching")
		assert.Equal(t, driverID, resp.DriverID, "expected matched driver to be the registered driver")
		assert.GreaterOrEqual(t, resp.ETA, 0, "expected valid ETA seconds")
	})

	// 5. Create Trip
	t.Run("CreateTrip", func(t *testing.T) {
		reqBody := CreateTripRequest{
			RiderID: riderID,
			Lat:     testLat,
			Lng:     testLng,
		}
		var resp TripResponse
		code, err := client.Request(ctx, "POST", "/api/trip/trips", nil, reqBody, &resp)
		require.NoError(t, err, "failed to create trip")
		assert.Equal(t, 201, code, "expected HTTP 201 Created for trip creation")
		assert.NotEmpty(t, resp.ID, "expected non-empty trip ID")
		assert.Equal(t, riderID, resp.RiderID)
		assert.Equal(t, "requested", resp.State, "expected initial trip state to be 'requested'")
		tripID = resp.ID
	})

	// 6. Accept Trip
	t.Run("AcceptTrip", func(t *testing.T) {
		require.NotEmpty(t, tripID, "cannot run AcceptTrip without trip ID")

		reqBody := AcceptTripRequest{
			DriverID: driverID,
		}
		code, err := client.Request(ctx, "POST", fmt.Sprintf("/api/trip/trips/%s/accept", tripID), nil, reqBody, nil)
		require.NoError(t, err, "failed to accept trip")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for trip acceptance")

		// Verify state via GET
		var resp TripResponse
		getCode, getErr := client.Request(ctx, "GET", fmt.Sprintf("/api/trip/trips/%s", tripID), nil, nil, &resp)
		require.NoError(t, getErr, "failed to retrieve trip details")
		assert.Equal(t, 200, getCode)
		assert.Equal(t, "accepted", resp.State, "expected trip state to be 'accepted'")
		require.NotNil(t, resp.DriverID)
		assert.Equal(t, driverID, *resp.DriverID, "expected driver ID to be populated")
	})

	// 7. Start Trip
	t.Run("StartTrip", func(t *testing.T) {
		require.NotEmpty(t, tripID, "cannot run StartTrip without trip ID")

		code, err := client.Request(ctx, "POST", fmt.Sprintf("/api/trip/trips/%s/start", tripID), nil, nil, nil)
		require.NoError(t, err, "failed to start trip")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for starting trip")

		// Verify state via GET
		var resp TripResponse
		getCode, getErr := client.Request(ctx, "GET", fmt.Sprintf("/api/trip/trips/%s", tripID), nil, nil, &resp)
		require.NoError(t, getErr, "failed to retrieve trip details")
		assert.Equal(t, 200, getCode)
		assert.Equal(t, "in-progress", resp.State, "expected trip state to be 'in-progress'")
	})

	// 8. Complete Trip
	t.Run("CompleteTrip", func(t *testing.T) {
		require.NotEmpty(t, tripID, "cannot run CompleteTrip without trip ID")

		code, err := client.Request(ctx, "POST", fmt.Sprintf("/api/trip/trips/%s/complete", tripID), nil, nil, nil)
		require.NoError(t, err, "failed to complete trip")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for completing trip")

		// Verify state via GET
		var resp TripResponse
		getCode, getErr := client.Request(ctx, "GET", fmt.Sprintf("/api/trip/trips/%s", tripID), nil, nil, &resp)
		require.NoError(t, getErr, "failed to retrieve trip details")
		assert.Equal(t, 200, getCode)
		assert.Equal(t, "completed", resp.State, "expected trip state to be 'completed'")
	})

	// 9. Create Payment
	t.Run("CreatePayment", func(t *testing.T) {
		require.NotEmpty(t, tripID, "cannot run CreatePayment without trip ID")

		reqBody := CreatePaymentRequest{
			TripID: tripID,
			Amount: 2500, // $25.00
		}
		var resp PaymentResponse
		code, err := client.Request(ctx, "POST", "/api/billing/payments", nil, reqBody, &resp)
		require.NoError(t, err, "failed to create payment")
		assert.Equal(t, 201, code, "expected HTTP 201 Created for payment creation")
		assert.NotEmpty(t, resp.ID, "expected payment ID to be populated")
		assert.Equal(t, tripID, resp.TripID)
		assert.Equal(t, int64(2500), resp.Amount)
		assert.Equal(t, "pending", resp.State, "expected initial payment state to be 'pending'")
		paymentID = resp.ID
	})

	// 10. Process Payment
	t.Run("ProcessPayment", func(t *testing.T) {
		require.NotEmpty(t, paymentID, "cannot run ProcessPayment without payment ID")

		code, err := client.Request(ctx, "POST", fmt.Sprintf("/api/billing/payments/%s/process", paymentID), nil, nil, nil)
		require.NoError(t, err, "failed to process payment")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for processing payment")
	})

	// 11. Submit Review
	t.Run("SubmitReview", func(t *testing.T) {
		require.NotEmpty(t, tripID, "cannot run SubmitReview without trip ID")

		reqBody := CreateReviewRequest{
			TripID:    tripID,
			SubjectID: driverID,
			Score:     5,
			Comment:   "Perfect ride!",
		}
		headers := map[string]string{
			"X-User-ID": riderID,
		}
		var resp ReviewResponse
		code, err := client.Request(ctx, "POST", "/api/review/reviews", headers, reqBody, &resp)
		require.NoError(t, err, "failed to submit review")
		assert.Equal(t, 201, code, "expected HTTP 201 Created for review creation")
		assert.NotEmpty(t, resp.ID, "expected review ID to be populated")
		assert.Equal(t, tripID, resp.TripID)
		assert.Equal(t, riderID, resp.ReviewerID)
		assert.Equal(t, driverID, resp.SubjectID)
		assert.Equal(t, 5, resp.Score)
		assert.Equal(t, "Perfect ride!", resp.Comment)
	})

	// 12. Verify Driver Rating
	t.Run("VerifyDriverRating", func(t *testing.T) {
		var resp RatingResponse
		code, err := client.Request(ctx, "GET", fmt.Sprintf("/api/review/users/%s/rating", driverID), nil, nil, &resp)
		require.NoError(t, err, "failed to retrieve driver rating")
		assert.Equal(t, 200, code, "expected HTTP 200 OK for driver rating retrieval")
		assert.Equal(t, 5.0, resp.Average, "expected average driver rating to be 5.0")
	})
}
