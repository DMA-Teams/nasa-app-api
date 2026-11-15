package agro

import (
	"math"
	"testing"
	"time"
)

func TestMonthlyPrecipitationOutliers_FlagsPowerAnomalyPattern(t *testing.T) {
	t.Parallel()

	start := time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)
	days := int(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Sub(start).Hours() / 24)
	s := &CellSeries{Start: start, Precipitation: make([]Millimeters, days)}

	januaryTotals := map[int]float64{2021: 300, 2022: 250}

	for i := range days {
		d := start.AddDate(0, 0, i)
		monthly := 20.0 + float64(d.Year()%5)

		if v, ok := januaryTotals[d.Year()]; ok && d.Month() == time.January {
			monthly = v
		}

		s.Precipitation[i] = Millimeters(monthly / 31)
	}

	s.Precipitation[100] = Millimeters(math.NaN())

	got := MonthlyPrecipitationOutliers(s)

	want := []YearMonth{{2021, time.January}, {2022, time.January}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("outliers = %v, want %v", got, want)
	}
}
