package agro

import (
	"math"
	"slices"
	"time"
)

const (
	outlierIQRMultiplier = 3.0
	minimumYearsForQA    = 8
)

func MonthlyPrecipitationOutliers(s *CellSeries) []YearMonth {
	totals := map[YearMonth]float64{}

	for i := range s.Len() {
		d := s.Start.AddDate(0, 0, i)
		totals[YearMonth{d.Year(), d.Month()}] += float64(s.Precipitation[i])
	}

	var out []YearMonth

	for m := time.January; m <= time.December; m++ {
		var keys []YearMonth

		var values []float64

		for k, v := range totals {
			if k.Month == m && !math.IsNaN(v) {
				keys = append(keys, k)
				values = append(values, v)
			}
		}

		if len(values) < minimumYearsForQA {
			continue
		}

		q1, q3 := Percentile(values, 0.25), Percentile(values, 0.75)
		threshold := q3 + outlierIQRMultiplier*(q3-q1)

		for i, v := range values {
			if v > threshold {
				out = append(out, keys[i])
			}
		}
	}

	slices.SortFunc(out, func(a, b YearMonth) int {
		if a.Year != b.Year {
			return a.Year - b.Year
		}

		return int(a.Month) - int(b.Month)
	})

	return out
}
