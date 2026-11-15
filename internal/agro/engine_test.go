package agro

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func testCrops() []*Crop {
	maize := testMaize()

	beans := &Crop{
		ID: "beans", Name: "Frijol", Family: "Fabaceae", Role: RoleHarvest,
		StageDays: [4]int{15, 20, 30, 15}, KcIni: 0.4, KcMid: 1.15, KcEnd: 0.35,
		MaxRootDepth: 0.7, DepletionFraction: 0.45, KyTotal: 1.15,
		KyByPhase: [4]float64{0.2, 1.1, 0.75, 0.2}, HasKyByPhase: true,
		Temperature:    TemperatureRange{Base: 8, AbsoluteLo: 5, OptimalLo: 15, OptimalHi: 27, AbsoluteHi: 32},
		HeatStressTMax: 31, PH: PHRange{AbsoluteLo: 4.5, OptimalLo: 5.5, OptimalHi: 7, AbsoluteHi: 8.2},
		AltitudeMin: 200, AltitudeMax: 1800, Irrigation: IrrigationOptional, Waterlogging: WaterloggingSensitive,
		NitrogenFixationKgHa: 40, NitrogenRemovalKgPerTon: 35, ReferenceYieldTonHa: 0.8,
		CoverFactorByStage: [4]float64{0.7, 0.4, 0.2, 0.3}, RelayGuest: RelayGrain,
	}

	sorghum := &Crop{
		ID: "sorghum", Name: "Sorgo", Family: "Poaceae", Role: RoleHarvest,
		StageDays: [4]int{20, 35, 40, 30}, KcIni: 0.3, KcMid: 1.05, KcEnd: 0.55,
		MaxRootDepth: 1.5, DepletionFraction: 0.55, KyTotal: 0.9,
		KyByPhase: [4]float64{0.2, 0.55, 0.45, 0.2}, HasKyByPhase: true,
		Temperature:    TemperatureRange{Base: 10, AbsoluteLo: 12, OptimalLo: 22, OptimalHi: 35, AbsoluteHi: 40},
		HeatStressTMax: 38, PH: PHRange{AbsoluteLo: 4.3, OptimalLo: 5.5, OptimalHi: 7.5, AbsoluteHi: 8.7},
		AltitudeMin: 0, AltitudeMax: 1200, Irrigation: IrrigationOptional, Waterlogging: WaterloggingModerate,
		NitrogenRemovalKgPerTon: 15, ReferenceYieldTonHa: 1.5,
		CoverFactorByStage: [4]float64{0.8, 0.5, 0.25, 0.35}, RelayHosts: []RelayKind{RelayCover},
	}

	mucuna := &Crop{
		ID: "mucuna", Name: "Mucuna", Family: "Fabaceae", Role: RoleCover,
		StageDays: [4]int{20, 40, 60, 30}, KcIni: 0.4, KcMid: 1.1, KcEnd: 0.5,
		MaxRootDepth: 1.0, DepletionFraction: 0.5, KyTotal: 1,
		Temperature:    TemperatureRange{Base: 10, AbsoluteLo: 10, OptimalLo: 20, OptimalHi: 30, AbsoluteHi: 38},
		HeatStressTMax: 38, PH: PHRange{AbsoluteLo: 4, OptimalLo: 5, OptimalHi: 7, AbsoluteHi: 8},
		AltitudeMin: 0, AltitudeMax: 1500, Irrigation: IrrigationOptional, Waterlogging: WaterloggingModerate,
		NitrogenFixationKgHa: 150, CoverFactorByStage: [4]float64{0.6, 0.2, 0.05, 0.05}, RelayGuest: RelayCover,
		LowConfidenceFields: []string{"stages", "kc"},
	}

	rice := &Crop{
		ID: "rice", Name: "Arroz", Family: "Poaceae", Role: RoleHarvest,
		StageDays: [4]int{30, 30, 60, 30}, KcIni: 1.05, KcMid: 1.2, KcEnd: 0.75,
		MaxRootDepth: 0.75, DepletionFraction: 0.2, KyTotal: 1,
		Temperature:    TemperatureRange{Base: 10, AbsoluteLo: 10, OptimalLo: 20, OptimalHi: 32, AbsoluteHi: 38},
		HeatStressTMax: 35, PH: PHRange{AbsoluteLo: 4, OptimalLo: 5, OptimalHi: 7, AbsoluteHi: 8.5},
		AltitudeMin: 0, AltitudeMax: 1000, Irrigation: IrrigationRequired, Waterlogging: WaterloggingTolerant,
		NitrogenRemovalKgPerTon: 13, ReferenceYieldTonHa: 3.5,
		CoverFactorByStage: [4]float64{0.8, 0.5, 0.25, 0.35},
	}

	return []*Crop{maize, beans, sorghum, mucuna, rice}
}

func testTemplates() []Template {
	return []Template{
		{ID: "maize-beans", Name: "Maíz y frijol", Crops: []string{"maize", "beans"}, Relay: []bool{false, false}},
		{ID: "maize-sorghum", Name: "Maíz y sorgo", Crops: []string{"maize", "sorghum"}, Relay: []bool{false, false}},
		{ID: "maize-mucuna", Name: "Maíz con mucuna", Crops: []string{"maize", "mucuna"}, Relay: []bool{false, true}},
		{ID: "quesungual", Name: "Quesungual", Crops: []string{"maize", "beans"}, Relay: []bool{false, false}, Mulch: true},
	}
}

func dryCorridorRequest(t *testing.T, irrigated bool) EvaluationRequest {
	t.Helper()

	series := dryCorridorSeries(30)
	profile := &SiteProfile{
		MunicipalityCode: "TEST", ElevationMedian: 1000,
		Soil: SoilSummary{ClayPercent: 30, SandPercent: 35, SiltPercent: 35, OrganicCarbonGPerKg: 20, PH: 6.3},
	}
	profile.FillClimate(series)

	return EvaluationRequest{
		Start: time.Date(2027, time.January, 10, 0, 0, 0, 0, time.UTC), Series: series, Profile: profile,
		Irrigated: irrigated, Priority: PriorityBalanced, CurrentRotation: []string{"maize", "maize"},
	}
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()

	e, err := NewEngine(testCrops(), testTemplates())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	return e
}

func TestEvaluate_DryCorridorAcceptance(t *testing.T) {
	t.Parallel()

	ev, err := newTestEngine(t).Evaluate(context.Background(), dryCorridorRequest(t, false))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if len(ev.Rotations) < 3 || ev.Rotations[0].Kind != KindCurrent {
		t.Fatalf("got %d rotations, first %q; want >= 3 starting with the current one", len(ev.Rotations), ev.Rotations[0].Kind)
	}

	for _, r := range ev.Rotations {
		for _, p := range r.Plantings {
			if p.Role == "harvest" && (p.Season == "verano") {
				t.Errorf("rotation %s sows rainfed %s in %s (%s), want no dry-season sowing", r.ID, p.CropID, p.Season, p.SowDate)
			}

			if p.CropID == "rice" {
				t.Errorf("rotation %s uses rice without irrigation", r.ID)
			}
		}

		if len(r.Quarters) != 4 || len(r.Semesters) != 2 {
			t.Errorf("rotation %s: %d quarters %d semesters, want 4 and 2", r.ID, len(r.Quarters), len(r.Semesters))
		}
	}

	beansSeason := ""
	for _, r := range ev.Rotations {
		if r.ID == "maize-beans" && r.Feasible && len(r.Plantings) == 2 {
			beansSeason = r.Plantings[1].Season
		}
	}

	if beansSeason != "postrera" {
		t.Errorf("maize-beans template sows beans in %q, want postrera", beansSeason)
	}

	if ev.Best == "" {
		t.Error("no best rotation selected")
	}
}

func TestEvaluate_FamilyRuleAppliesToGeneratedRotations(t *testing.T) {
	t.Parallel()

	ev, err := newTestEngine(t).Evaluate(context.Background(), dryCorridorRequest(t, false))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	for _, r := range ev.Rotations {
		if r.Kind != KindGenerated {
			continue
		}

		for i := 1; i < len(r.Plantings); i++ {
			if r.Plantings[i].Family == r.Plantings[i-1].Family {
				t.Errorf("generated %s repeats family %s", r.Signature, r.Plantings[i].Family)
			}
		}
	}

	current := ev.Rotations[0]
	if current.Scores.Soil >= bestSoil(ev) {
		t.Errorf("maize after maize soil score %.1f should be below the best %.1f", current.Scores.Soil, bestSoil(ev))
	}
}

func bestSoil(ev Evaluation) float64 {
	best := 0.0
	for _, r := range ev.Rotations {
		best = max(best, r.Scores.Soil)
	}

	return best
}

func TestEvaluate_IrrigationEnablesRice(t *testing.T) {
	t.Parallel()

	req := dryCorridorRequest(t, true)
	req.CurrentRotation = []string{"rice"}

	ev, err := newTestEngine(t).Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	current := ev.Rotations[0]
	if !current.Feasible || current.Plantings[0].CropID != "rice" || current.Plantings[0].IrrigationMm <= 0 {
		t.Errorf("irrigated rice: %+v, want feasible with irrigation reported", current.Plantings)
	}
}

func TestEvaluate_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newTestEngine(t).Evaluate(ctx, dryCorridorRequest(t, false))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestEvaluate_ForecastMarksPeriods(t *testing.T) {
	t.Parallel()

	req := dryCorridorRequest(t, false)
	req.Forecast = make([]*TercileProbabilities, planningHorizonMonths)

	for k := range 6 {
		req.Forecast[k] = &TercileProbabilities{Below: 0.6, Normal: 0.3, Above: 0.1}
	}

	ev, err := newTestEngine(t).Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	r := ev.Rotations[0]
	if !r.Semesters[0].ForecastInfluenced || r.Semesters[1].ForecastInfluenced {
		t.Errorf("semesters forecast flags = %v %v, want true false", r.Semesters[0].ForecastInfluenced, r.Semesters[1].ForecastInfluenced)
	}

	if ev.ForecastMonths != 6 {
		t.Errorf("forecast months = %d, want 6", ev.ForecastMonths)
	}
}

func TestEvaluate_RejectsShortHistory(t *testing.T) {
	t.Parallel()

	req := dryCorridorRequest(t, false)
	req.Series = dryCorridorSeries(5)

	_, err := newTestEngine(t).Evaluate(context.Background(), req)
	if !errors.Is(err, ErrInsufficientHistory) {
		t.Errorf("err = %v, want ErrInsufficientHistory", err)
	}
}

func BenchmarkEvaluate(b *testing.B) {
	t := &testing.T{}
	e, _ := NewEngine(testCrops(), testTemplates())
	req := dryCorridorRequest(t, false)

	for b.Loop() {
		if _, err := e.Evaluate(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

func TestFits_RelayRules(t *testing.T) {
	t.Parallel()

	crops := map[string]*Crop{}
	for _, c := range testCrops() {
		crops[c.ID] = c
	}

	host := func(id string) *plantingStats {
		c := crops[id]

		return &plantingStats{crop: c, sow: 0, stages: c.StageDays, cycle: c.CycleDays()}
	}
	guest := func(id string, sow int) *plantingStats {
		c := crops[id]

		return &plantingStats{crop: c, sow: sow, stages: c.StageDays, cycle: c.CycleDays()}
	}

	tests := []struct {
		name  string
		host  string
		guest string
		sow   int
		want  bool
	}{
		{"cover into maize after 30 days", "maize", "mucuna", 30, true},
		{"cover into maize too early", "maize", "mucuna", 29, false},
		{"cover into maize after harvest", "maize", "mucuna", 125, false},
		{"cover into sorghum", "sorghum", "mucuna", 45, true},
		{"cover into beans", "beans", "mucuna", 40, false},
		{"beans into maize at maturity", "maize", "beans", 95, true},
		{"beans into maize before maturity", "maize", "beans", 94, false},
		{"beans into sorghum", "sorghum", "beans", 100, false},
		{"rice is not a host", "rice", "mucuna", 40, false},
	}

	ev := &evaluator{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ev.fits([]*plantingStats{host(tt.host)}, guest(tt.guest, tt.sow), true, false); got != tt.want {
				t.Errorf("fits = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEngine_RejectsRelayOnNonHost(t *testing.T) {
	t.Parallel()

	_, err := NewEngine(testCrops(), []Template{{ID: "bad", Name: "bad", Crops: []string{"beans", "mucuna"}, Relay: []bool{false, true}}})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
}

func TestParsePriority_IncomeIsOutOfScope(t *testing.T) {
	t.Parallel()

	if _, ok := ParsePriority("income"); ok {
		t.Error("income priority accepted, want rejected")
	}

	for _, p := range []Priority{PriorityBalanced, PriorityWater, PrioritySoil} {
		if w, s := p.weights(); math.Abs(w+s-1) > 1e-9 {
			t.Errorf("priority %d weights %.2f + %.2f, want sum 1", p, w, s)
		}
	}
}
