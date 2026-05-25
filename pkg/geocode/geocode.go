package geocode

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"haillion/pkg/geocode/data"
)

// Location represents a geographic point with Latitude and Longitude.
type Location struct {
	Lat float64
	Lng float64
}

var (
	// ErrInvalidPostcode is returned when a postcode is empty or malformed.
	ErrInvalidPostcode = errors.New("invalid postcode")
	// ErrPostcodeNotFound is returned when a valid outward code is not found in the dataset.
	ErrPostcodeNotFound = errors.New("postcode not found")
)

// regex for UK outward codes validation.
// Pattern: 1 or 2 letters, followed by a digit, optionally followed by another digit or letter.
var outwardRegex = regexp.MustCompile(`^[A-Z]{1,2}[0-9][0-9A-Z]?$`)

// centroids holds the loaded outward postcode centroid coordinates.
var centroids map[string]Location

func init() {
	centroids = make(map[string]Location)

	reader := csv.NewReader(bytes.NewReader(data.CentroidsCSV))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Panic is acceptable in package init if the embedded required resource is corrupt.
			panic(fmt.Sprintf("failed to parse embedded postcode dataset: %v", err))
		}

		if len(record) < 3 {
			continue
		}

		outward := strings.ToUpper(strings.TrimSpace(record[0]))
		latVal, err1 := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		lngVal, err2 := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)

		if err1 == nil && err2 == nil {
			centroids[outward] = Location{
				Lat: latVal,
				Lng: lngVal,
			}
		}
	}
}

// LookupOutward resolves a UK outward code (e.g. "SW1A") to a centroid.
// Accepts full postcodes ("SW1A 1AA") by extracting the outward portion.
// Case-insensitive; whitespace trimmed.
func LookupOutward(ctx context.Context, postcode string) (Location, error) {
	normalised, err := Normalise(postcode)
	if err != nil {
		return Location{}, err
	}

	loc, ok := centroids[normalised]
	if !ok {
		return Location{}, ErrPostcodeNotFound
	}

	return loc, nil
}

// Normalise cleans and normalises the postcode input to extract a valid outward code.
// Rules:
// 1. Trim leading and trailing whitespace.
// 2. If there is a space in the trimmed string, take the portion before the first space.
// 3. If there is no space, but length is > 4, check if the last 3 characters resemble
//    the standard UK inward format (digit + letter + letter). If so, extract the outward portion.
// 4. Uppercase.
// 5. Validate against the outward code regex `^[A-Z]{1,2}[0-9][0-9A-Z]?$`.
func Normalise(postcode string) (string, error) {
	trimmed := strings.TrimSpace(postcode)
	if trimmed == "" {
		return "", ErrInvalidPostcode
	}

	var outward string
	if idx := strings.Index(trimmed, " "); idx != -1 {
		outward = trimmed[:idx]
	} else if len(trimmed) > 4 {
		// Attempt to split inward code if no space is present but length suggests a full postcode.
		// A standard UK inward postcode is 3 characters: a digit followed by two letters.
		inwardStart := len(trimmed) - 3
		if isInwardCode(trimmed[inwardStart:]) {
			outward = trimmed[:inwardStart]
		} else {
			outward = trimmed
		}
	} else {
		outward = trimmed
	}

	outward = strings.ToUpper(strings.TrimSpace(outward))

	if !outwardRegex.MatchString(outward) {
		return "", ErrInvalidPostcode
	}

	return outward, nil
}

// isInwardCode checks if the 3-character string matches the UK inward postcode format (digit + letter + letter).
func isInwardCode(s string) bool {
	if len(s) != 3 {
		return false
	}
	r := []rune(s)
	// First character must be a digit (0-9)
	if r[0] < '0' || r[0] > '9' {
		return false
	}
	// Second and third must be letters (A-Z, a-z)
	if !isLetter(r[1]) || !isLetter(r[2]) {
		return false
	}
	return true
}

func isLetter(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

// Haversine returns the great-circle distance between two Locations in miles.
func Haversine(a, b Location) float64 {
	if a.Lat == b.Lat && a.Lng == b.Lng {
		return 0.0
	}

	const r = 3958.7613 // Earth mean radius in miles

	lat1 := deg2rad(a.Lat)
	lng1 := deg2rad(a.Lng)
	lat2 := deg2rad(b.Lat)
	lng2 := deg2rad(b.Lng)

	dlat := lat2 - lat1
	dlng := lng2 - lng1

	h := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlng/2)*math.Sin(dlng/2)

	// Clamp h to [0.0, 1.0] to avoid float precision overflow in math.Sqrt/math.Asin
	if h > 1.0 {
		h = 1.0
	} else if h < 0.0 {
		h = 0.0
	}

	return 2.0 * r * math.Asin(math.Sqrt(h))
}

func deg2rad(deg float64) float64 {
	return deg * math.Pi / 180.0
}
