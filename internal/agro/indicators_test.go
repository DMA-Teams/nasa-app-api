package agro

import (
	"math"
	"testing"
	"time"
)

func dryCorridorSeries(years int) *SiteSeries {
	monthlyMm := [12]float64{25, 17, 15, 30, 126, 197, 138, 173, 206, 154, 54, 34}
	start := time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(years, 0, 0)
	days := int(end.Sub(start).Hours() / 24)
	s := &SiteSeries{
		Start: start, Precipitation: make([]Millimeters, days), ET0: make([]Millimeters, days),
		TMean: make([]Celsius, days), TMax: make([]Celsius, days), TMin: make([]Celsius, days),
	}

	for i := range days {
		d := start.AddDate(0, 0, i)
		m := d.Month()
		daysInMonth := time.Date(d.Year(), m+1, 0, 0, 0, 0, 0, time.UTC).Day()
		rain := monthlyMm[m-1] / float64(daysInMonth) * 2
		if d.Day()%2 == 0 {
			rain = 0
		}

		if m == time.July && d.Day() >= 5 && d.Day() <= 28 {
			rain = 0
		}

		s.Precipitation[i] = Millimeters(rain)
		s.ET0[i] = Millimeters(4.2 + 0.8*math.Sin(2*math.Pi*float64(d.YearDay()-60)/365))
		s.TMean[i] = 23
		s.TMax[i] = 30
		s.TMin[i] = 17
	}

	return s
}

func TestPercentile(t *testing.T) {
	t.Parallel()

	values := []float64{4, 1, 3, 2, math.NaN(), 5}
	assertNear(t, "P50", Percentile(values, 0.5), 3, 1e-9)
	assertNear(t, "P0", Percentile(values, 0), 1, 1e-9)
	assertNear(t, "P100", Percentile(values, 1), 5, 1e-9)

	if !math.IsNaN(Percentile(nil, 0.5)) {
		t.Error("Percentile of empty slice must be NaN")
	}
}

func TestOnsetOfRains_DryCorridorStartsInMay(t *testing.T) {
	t.Parallel()

	s := dryCorridorSeries(3)

	doy, ok := OnsetOfRains(s, 1997)
	if !ok {
		t.Fatal("onset not found")
	}

	if doy < 115 || doy > 160 {
		t.Errorf("onset day of year = %d, want around May (115-160)", doy)
	}
}

func TestCanicula_DetectedInJuly(t *testing.T) {
	t.Parallel()

	s := dryCorridorSeries(3)

	w, ok := Canicula(s, 1997)
	if !ok {
		t.Fatal("canícula not detected")
	}

	if w.StartDOY < 175 || w.StartDOY > 200 || w.Length() < 15 {
		t.Errorf("canícula = %+v, want a window starting in July lasting >= 15 days", w)
	}
}

func TestFortnightOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		date time.Time
		want int
	}{
		{time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC), 0},
		{time.Date(2027, time.January, 16, 0, 0, 0, 0, time.UTC), 1},
		{time.Date(2027, time.December, 31, 0, 0, 0, 0, time.UTC), 23},
	}

	for _, tt := range tests {
		if got := FortnightOf(tt.date); got != tt.want {
			t.Errorf("FortnightOf(%s) = %d, want %d", tt.date.Format(time.DateOnly), got, tt.want)
		}
	}
}

func TestFortnightClimatology_DrySeasonHasDeficit(t *testing.T) {
	t.Parallel()

	f := FortnightClimatology(dryCorridorSeries(5))

	if f[2].DeficitMedian >= 0 {
		t.Errorf("February deficit = %.1f, want < 0", f[2].DeficitMedian)
	}

	if f[17].DeficitMedian <= f[2].DeficitMedian {
		t.Errorf("September deficit %.1f should exceed February %.1f", f[17].DeficitMedian, f[2].DeficitMedian)
	}
}

func TestAnalogMonthWeights_SumToOneAndFavorForecastTercile(t *testing.T) {
	t.Parallel()

	totals := [][]float64{{10}, {20}, {30}, {40}, {50}, {60}}
	dry := &TercileProbabilities{Below: 0.7, Normal: 0.2, Above: 0.1}
	w := AnalogMonthWeights(totals, []*TercileProbabilities{dry})

	sum := 0.0
	for a := range w {
		sum += w[a][0]
	}

	assertNear(t, "sum", sum, 1, 1e-9)

	if w[0][0] <= w[5][0] {
		t.Errorf("dry analog weight %.3f should exceed wet analog weight %.3f", w[0][0], w[5][0])
	}
}

func TestAnalogMonthWeights_UniformWithoutForecastAndZeroForMissing(t *testing.T) {
	t.Parallel()

	totals := [][]float64{{10}, {math.NaN()}, {30}, {40}}
	w := AnalogMonthWeights(totals, []*TercileProbabilities{nil})

	if w[1][0] != 0 {
		t.Errorf("missing analog weight = %.3f, want 0", w[1][0])
	}

	assertNear(t, "uniform", w[0][0], 1.0/3, 1e-9)
}

func TestPlantingWeights_Normalized(t *testing.T) {
	t.Parallel()

	monthWeights := [][]float64{{0.5, 0.2}, {0.5, 0.8}}
	w := PlantingWeights(monthWeights, []int{0, 1}, []bool{true, true})
	assertNear(t, "sum", w[0]+w[1], 1, 1e-9)

	w = PlantingWeights(monthWeights, []int{0, 1}, []bool{false, true})
	assertNear(t, "only valid", w[1], 1, 1e-9)
}

func TestSuitability_LimitingFactor(t *testing.T) {
	t.Parallel()

	c := testMaize()

	score, limit := Suitability(c, SiteConditions{MeanTemperature: 24, PH: 6.3, ClayPercent: 30, Elevation: 1000})
	if score != 1 || limit != LimitNone {
		t.Errorf("optimal site: score %.2f limit %q, want 1 and none", score, limit)
	}

	_, limit = Suitability(c, SiteConditions{MeanTemperature: 24, PH: 4.0, ClayPercent: 30, Elevation: 1000})
	if limit != LimitPH {
		t.Errorf("acid soil limit = %q, want ph", limit)
	}

	c.Irrigation = IrrigationRequired
	score, limit = Suitability(c, SiteConditions{MeanTemperature: 24, PH: 6.3, ClayPercent: 30, Elevation: 1000})
	if score != 0 || limit != LimitIrrigation {
		t.Errorf("rainfed site for irrigated crop: %.2f %q, want 0 irrigation", score, limit)
	}
}

func TestCaniculaRule_StricterRulesDetectNoMoreThanLooserOnes(t *testing.T) {
	t.Parallel()

	s := dryCorridorSeries(3)

	w1, ok1 := Canicula(s, 1997)
	w2, ok2 := DefaultCaniculaRule.Detect(s, 1997)

	if w1 != w2 || ok1 != ok2 {
		t.Fatalf("Canicula = %+v %v, DefaultCaniculaRule.Detect = %+v %v", w1, ok1, w2, ok2)
	}

	loose := CaniculaRule{DeficitRatio: 0.8, MinimumDays: 3}
	strict := CaniculaRule{DeficitRatio: 0.2, MinimumDays: 30}

	wl, okl := loose.Detect(s, 1997)
	ws, oks := strict.Detect(s, 1997)

	if oks && !okl {
		t.Errorf("strict rule detected %+v but loose rule did not", ws)
	}

	if oks && ws.Length() > wl.Length() {
		t.Errorf("strict window %+v longer than loose window %+v", ws, wl)
	}

	if _, ok := (CaniculaRule{DeficitRatio: 0, MinimumDays: 1}).Detect(s, 1997); ok {
		t.Error("a zero deficit ratio must never detect a canícula")
	}
}
