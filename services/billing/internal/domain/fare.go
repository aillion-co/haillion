package domain

import (
	"math"
)

const (
	PerMileRatePence = 100
	MinFarePence     = 250
)

type FareBreakdown struct {
	DistanceMiles   float64 `json:"distance_miles"`
	BasePence       int64   `json:"base_pence"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	TotalPence      int64   `json:"total_pence"`
	MinimumApplied  bool    `json:"minimum_applied"`
}

// EstimateFare computes a fare for the given distance and supply/demand.
// Inputs: distance in miles, active rider count, active driver count.
// Pure function — no I/O.
func EstimateFare(distanceMiles float64, activeRiders, activeDrivers int) FareBreakdown {
	// Round distance to 2 decimal places
	roundedDistance := math.Round(distanceMiles*100) / 100

	basePence := int64(math.Round(roundedDistance * PerMileRatePence))
	surgeMultiplier := CalculateSurge(activeRiders, activeDrivers)

	rawTotal := float64(basePence) * surgeMultiplier
	totalPenceUncapped := int64(math.Round(rawTotal))

	minimumApplied := totalPenceUncapped < MinFarePence
	totalPence := totalPenceUncapped
	if totalPence < MinFarePence {
		totalPence = MinFarePence
	}

	return FareBreakdown{
		DistanceMiles:   roundedDistance,
		BasePence:       basePence,
		SurgeMultiplier: surgeMultiplier,
		TotalPence:      totalPence,
		MinimumApplied:  minimumApplied,
	}
}
