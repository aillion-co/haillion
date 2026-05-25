package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimateFare(t *testing.T) {
	tests := []struct {
		name           string
		distanceMiles  float64
		riders         int
		drivers        int
		expectedDist   float64
		expectedBase   int64
		expectedSurge  float64
		expectedTotal  int64
		expectedMinApp bool
	}{
		{
			name:           "zero distance",
			distanceMiles:  0.0,
			riders:         0,
			drivers:        0,
			expectedDist:   0.0,
			expectedBase:   0,
			expectedSurge:  1.0,
			expectedTotal:  250,
			expectedMinApp: true,
		},
		{
			name:           "1.5 mi x surge 1.0",
			distanceMiles:  1.5,
			riders:         5,
			drivers:        5,
			expectedDist:   1.5,
			expectedBase:   150,
			expectedSurge:  1.0,
			expectedTotal:  250, // 150 < 250 so minimum applied
			expectedMinApp: true,
		},
		{
			name:           "1.5 mi x surge 2.0",
			distanceMiles:  1.5,
			riders:         10,
			drivers:        0,
			expectedDist:   1.5,
			expectedBase:   150,
			expectedSurge:  2.0,
			expectedTotal:  300, // 150 * 2.0 = 300 >= 250
			expectedMinApp: false,
		},
		{
			name:           "distance that rounds the base up",
			distanceMiles:  1.555,
			riders:         0,
			drivers:        0,
			expectedDist:   1.56,
			expectedBase:   156,
			expectedSurge:  1.0,
			expectedTotal:  250,
			expectedMinApp: true,
		},
		{
			name:           "distance that rounds the base down",
			distanceMiles:  1.554,
			riders:         0,
			drivers:        0,
			expectedDist:   1.55,
			expectedBase:   155,
			expectedSurge:  1.0,
			expectedTotal:  250,
			expectedMinApp: true,
		},
		{
			name:           "distance + surge where total < 250",
			distanceMiles:  1.2,
			riders:         10,
			drivers:        10,
			expectedDist:   1.2,
			expectedBase:   120,
			expectedSurge:  1.0,
			expectedTotal:  250,
			expectedMinApp: true,
		},
		{
			name:           "higher distance above minimum with surge",
			distanceMiles:  3.5,
			riders:         10,
			drivers:        5, // ratio 2.0 -> surge = 1.0 + 1.0*0.2 = 1.2
			expectedDist:   3.5,
			expectedBase:   350,
			expectedSurge:  1.2,
			expectedTotal:  420, // 350 * 1.2 = 420
			expectedMinApp: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateFare(tc.distanceMiles, tc.riders, tc.drivers)
			assert.Equal(t, tc.expectedDist, got.DistanceMiles)
			assert.Equal(t, tc.expectedBase, got.BasePence)
			assert.InDelta(t, tc.expectedSurge, got.SurgeMultiplier, 0.0001)
			assert.Equal(t, tc.expectedTotal, got.TotalPence)
			assert.Equal(t, tc.expectedMinApp, got.MinimumApplied)
		})
	}
}
