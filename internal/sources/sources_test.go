package sources

import (
	"math"
	"testing"
	"time"
)

const powerFixture = `{
  "geometry": {"type": "Point", "coordinates": [-88.6, 14.3, 1021.41]},
  "properties": {"parameter": {
    "PRECTOTCORR": {"20250601": 19.62, "20250602": -999.0},
    "T2M": {"20250601": 24.32, "20250602": 24.48},
    "T2M_MAX": {"20250601": 30.1, "20250602": 31.0},
    "T2M_MIN": {"20250601": 19.0, "20250602": 19.5},
    "RH2M": {"20250601": 85.0, "20250602": 80.0},
    "T2MDEW": {"20250601": 21.0, "20250602": 20.5},
    "WS2M": {"20250601": 1.5, "20250602": 1.7},
    "ALLSKY_SFC_SW_DWN": {"20250601": 17.2, "20250602": 19.0},
    "PS": {"20250601": 89.9, "20250602": 89.8}
  }},
  "header": {"fill_value": -999.0, "sources": ["MERRA2", "POWER"]}
}`

func TestParsePowerDaily_MapsFillValuesToNaN(t *testing.T) {
	t.Parallel()

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	got, err := ParsePowerDaily([]byte(powerFixture), from, from.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("ParsePowerDaily: %v", err)
	}

	s := got.Series
	if s.Len() != 2 || got.Elevation != 1021.41 || len(got.Sources) != 2 {
		t.Fatalf("len %d elevation %v sources %v", s.Len(), got.Elevation, got.Sources)
	}

	if s.Precipitation[0] != 19.62 || !math.IsNaN(float64(s.Precipitation[1])) {
		t.Errorf("precipitation = %v, want [19.62 NaN]", s.Precipitation)
	}

	if err := s.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

const soilFixture = `{"properties": {"layers": [
  {"name": "clay", "unit_measure": {"d_factor": 10}, "depths": [
    {"label": "0-5cm", "values": {"mean": 359, "Q0.05": 42, "Q0.95": 721}},
    {"label": "5-15cm", "values": {"mean": 372, "Q0.05": 50, "Q0.95": 730}},
    {"label": "15-30cm", "values": {"mean": 415, "Q0.05": 60, "Q0.95": 760}},
    {"label": "30-60cm", "values": {"mean": 474, "Q0.05": 47, "Q0.95": 860}}]},
  {"name": "phh2o", "unit_measure": {"d_factor": 10}, "depths": [
    {"label": "0-5cm", "values": {"mean": 63}}, {"label": "5-15cm", "values": {"mean": 63}},
    {"label": "15-30cm", "values": {"mean": 63}}, {"label": "30-60cm", "values": {"mean": 64}}]}
]}}`

func TestParseSoilGrids_ScalesAndWeightsTopsoil(t *testing.T) {
	t.Parallel()

	p, err := ParseSoilGrids([]byte(soilFixture))
	if err != nil {
		t.Fatalf("ParseSoilGrids: %v", err)
	}

	want := (5*35.9 + 10*37.2 + 15*41.5) / 30
	if got := p.Topsoil("clay", func(v SoilValue) float64 { return v.Mean }); math.Abs(got-want) > 1e-9 {
		t.Errorf("topsoil clay = %.3f, want %.3f", got, want)
	}

	s := p.Summary()
	if math.Abs(s.PH-6.3) > 1e-9 || s.ClayP05 >= s.ClayPercent || s.ClayP95 <= s.ClayPercent {
		t.Errorf("summary = %+v, want pH 6.3 and P05 < mean < P95", s)
	}
}

func TestSeasonalForecast_MemberAnomaliesUseModelClimatology(t *testing.T) {
	t.Parallel()

	var dates []any
	members := map[string]any{}

	for d := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC); d.Month() == time.January; d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format(time.DateOnly))
	}

	for m := range 3 {
		values := make([]any, len(dates))
		for i := range values {
			values[i] = float64(m + 1)
		}

		members["precipitation_sum_member0"+string(rune('1'+m))] = values
	}

	members["time"] = dates
	anomaly := -10.0

	f, err := parseSeasonal(members, []string{"2027-01-01"}, []*float64{&anomaly})
	if err != nil {
		t.Fatalf("parseSeasonal: %v", err)
	}

	if months := f.FullMonths(); len(months) != 1 || months[0] != [2]int{2027, 1} {
		t.Errorf("FullMonths = %v, want January 2027", months)
	}

	got, ok := f.MemberAnomalies(2027, 1)
	if !ok || len(got) != 3 {
		t.Fatalf("MemberAnomalies = %v, %v", got, ok)
	}

	mean := (got[0] + got[1] + got[2]) / 3
	if math.Abs(mean-anomaly) > 1e-9 {
		t.Errorf("mean member anomaly = %.3f, want the published ensemble anomaly %.1f", mean, anomaly)
	}
}
