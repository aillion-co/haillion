package data

import _ "embed"

// CentroidsCSV contains the raw CSV data for UK outward postcode centroids.
//
//go:embed uk_outward_centroids.csv
var CentroidsCSV []byte
var _ = CentroidsCSV
