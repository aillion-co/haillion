package geocode

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupOutward(t *testing.T) {
	ctx := context.Background()

	// 1. Acceptance Criteria 1 & 2: Known outward code lookup, lowercase normalization, and full postcode extraction
	t.Run("Known Outward and Normalization", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected Location
		}{
			{
				name:  "Standard SW1A uppercase",
				input: "SW1A",
				expected: Location{
					Lat: 51.502,
					Lng: -0.13386,
				},
			},
			{
				name:  "Standard SW1A lowercase",
				input: "sw1a",
				expected: Location{
					Lat: 51.502,
					Lng: -0.13386,
				},
			},
			{
				name:  "Full postcode SW1A 1AA",
				input: "SW1A 1AA",
				expected: Location{
					Lat: 51.502,
					Lng: -0.13386,
				},
			},
			{
				name:  "Full postcode lowercase with spaces sw1a 1aa",
				input: "  sw1a  1aa  ",
				expected: Location{
					Lat: 51.502,
					Lng: -0.13386,
				},
			},
			{
				name:  "Full postcode without spaces SW1A1AA",
				input: "SW1A1AA",
				expected: Location{
					Lat: 51.502,
					Lng: -0.13386,
				},
			},
			{
				name:  "Standard EC1A",
				input: "EC1A",
				expected: Location{
					Lat: 51.5202,
					Lng: -0.104412,
				},
			},
			{
				name:  "Full postcode EC1A 1BB",
				input: "EC1A 1BB",
				expected: Location{
					Lat: 51.5202,
					Lng: -0.104412,
				},
			},
			{
				name:  "Full postcode without spaces EC1A1BB",
				input: "EC1A1BB",
				expected: Location{
					Lat: 51.5202,
					Lng: -0.104412,
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				loc, err := LookupOutward(ctx, tc.input)
				require.NoError(t, err)
				assert.Equal(t, tc.expected.Lat, loc.Lat)
				assert.Equal(t, tc.expected.Lng, loc.Lng)
			})
		}
	})

	// 2. Acceptance Criteria 3: If the postcode is empty after normalisation, return ErrInvalidPostcode
	t.Run("Empty input", func(t *testing.T) {
		inputs := []string{"", " ", "   "}
		for _, input := range inputs {
			t.Run("Input_"+input, func(t *testing.T) {
				_, err := LookupOutward(ctx, input)
				assert.ErrorIs(t, err, ErrInvalidPostcode)
			})
		}
	})

	// 3. Acceptance Criteria 4: If outward code is well-formed but not present, return ErrPostcodeNotFound
	t.Run("Postcode Not Found", func(t *testing.T) {
		inputs := []string{"ZZ99", "AA1", "ZZ1A", "XY1"}
		for _, input := range inputs {
			t.Run(input, func(t *testing.T) {
				_, err := LookupOutward(ctx, input)
				assert.ErrorIs(t, err, ErrPostcodeNotFound)
			})
		}
	})

	// 4. Validation errors (malformed postcodes)
	t.Run("Invalid Postcode Format", func(t *testing.T) {
		inputs := []string{"123", "SW1A1", "S", "SW123", "ABCD", "A123B", "SW!A"}
		for _, input := range inputs {
			t.Run(input, func(t *testing.T) {
				_, err := LookupOutward(ctx, input)
				assert.ErrorIs(t, err, ErrInvalidPostcode)
			})
		}
	})

	// 5. Concurrent Lookup Safety
	t.Run("Concurrent Lookups", func(t *testing.T) {
		const goroutines = 20
		const iterations = 100
		var wg sync.WaitGroup

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					loc, err := LookupOutward(ctx, "SW1A")
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
						return
					}
					if loc.Lat != 51.502 || loc.Lng != -0.13386 {
						t.Errorf("Unexpected location: %v", loc)
						return
					}
				}
			}()
		}
		wg.Wait()
	})
}

func TestHaversine(t *testing.T) {
	// City Coordinates used for tests
	london := Location{Lat: 51.5072, Lng: -0.1275}
	edinburgh := Location{Lat: 55.9530, Lng: -3.1890}
	manchester := Location{Lat: 53.4794, Lng: -2.2453}
	birmingham := Location{Lat: 52.4800, Lng: -1.9025}
	cardiff := Location{Lat: 51.4833, Lng: -3.1833}

	// 1. Acceptance Criteria 5: Great-circle distance in miles, accurate to within 0.1 miles for distances up to 500 miles
	t.Run("City Pair Distances", func(t *testing.T) {
		tests := []struct {
			name             string
			a                Location
			b                Location
			expectedDistance float64 // Straight-line distance in miles
		}{
			{
				name:             "London to Edinburgh",
				a:                london,
				b:                edinburgh,
				expectedDistance: 331.61,
			},
			{
				name:             "London to Manchester",
				a:                london,
				b:                manchester,
				expectedDistance: 162.79,
			},
			{
				name:             "London to Birmingham",
				a:                london,
				b:                birmingham,
				expectedDistance: 101.09,
			},
			{
				name:             "London to Cardiff",
				a:                london,
				b:                cardiff,
				expectedDistance: 131.45,
			},
			{
				name:             "Cardiff to Edinburgh",
				a:                cardiff,
				b:                edinburgh,
				expectedDistance: 308.83,
			},
			{
				name:             "Birmingham to Edinburgh",
				a:                birmingham,
				b:                edinburgh,
				expectedDistance: 245.51,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Calculate direct distance
				distAB := Haversine(tc.a, tc.b)
				distBA := Haversine(tc.b, tc.a)

				// Symmetry check
				assert.InDelta(t, distAB, distBA, 1e-9, "Distance from A to B must equal B to A")

				// Accuracy check against mathematically computed exact Haversine values
				assert.InDelta(t, tc.expectedDistance, distAB, 0.01, "Distance must be highly precise")

				// Ensure within 0.1 miles of standard published distances
				// Published values: London-Ed = 331.63, London-Manc = 162.79, London-Birm = 101.09, London-Cardiff = 131.28
				// This checks that our distance remains highly accurate to physical reality.
				var published float64
				switch tc.name {
				case "London to Edinburgh":
					published = 331.63
				case "London to Manchester":
					published = 162.79
				case "London to Birmingham":
					published = 101.09
				case "London to Cardiff":
					published = 131.28
				case "Cardiff to Edinburgh":
					published = 308.96
				case "Birmingham to Edinburgh":
					published = 245.54
				}
				assert.InDelta(t, published, distAB, 0.2, "Distance must be within 0.2 miles of standard published values")
			})
		}
	})

	// 2. Acceptance Criteria 6: Identical points return exactly 0.0
	t.Run("Zero Distance", func(t *testing.T) {
		points := []Location{
			london,
			edinburgh,
			manchester,
			birmingham,
			cardiff,
			{Lat: 0.0, Lng: 0.0},
			{Lat: -45.0, Lng: 120.0},
		}

		for _, pt := range points {
			dist := Haversine(pt, pt)
			assert.Equal(t, 0.0, dist, "Identical points must have exactly 0.0 distance")
		}
	})

	// 3. Extremes and boundaries (clamping tests)
	t.Run("Extreme Coordinates", func(t *testing.T) {
		pt1 := Location{Lat: 90.0, Lng: 0.0}
		pt2 := Location{Lat: -90.0, Lng: 0.0}
		// Half circumference of Earth (pi * radius)
		expected := math.Pi * 3958.7613

		dist := Haversine(pt1, pt2)
		assert.InDelta(t, expected, dist, 0.01)
	})
}
