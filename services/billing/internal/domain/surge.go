package domain

func CalculateSurge(activeRiders, activeDrivers int) float64 {
	if activeDrivers <= 0 {
		if activeRiders > 0 {
			return 2.0 // High surge fallback when no drivers exist but riders are waiting
		}
		return 1.0 // Standard price if no drivers and no riders
	}

	ratio := float64(activeRiders) / float64(activeDrivers)
	if ratio > 1.0 {
		multiplier := 1.0 + (ratio-1.0)*0.2
		// Enforce a maximum surge multiplier of 3.0 to keep prices bounded
		if multiplier > 3.0 {
			return 3.0
		}
		return multiplier
	}

	return 1.0
}
