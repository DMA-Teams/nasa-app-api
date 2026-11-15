package agro

import (
	"fmt"
	"math"
	"time"
)

type plantingStats struct {
	crop            *Crop
	sow             int
	stages          [4]int
	cycle           int
	suitability     Fraction
	limit           LimitingFactor
	lossProbability float64
	expectedYield   float64
	medianYield     float64
	irrigationMm    float64
	heatDays        float64
	caniculaShare   float64
	months          []int
	analogsUsed     int
}

type plantingKey struct {
	crop  string
	sow   int
	mulch bool
}

type evaluator struct {
	engine         *Engine
	req            EvaluationRequest
	analogs        []int
	monthWeights   [][]float64
	forecastMask   [planningHorizonMonths]bool
	forecastMonths int
	meanTemp       []float64
	dailyBalance   []float64
	canicula       map[int]DayWindow
	availableWater Millimeters
	currentFamily  string
	scoring        scoringContext
	cache          map[plantingKey]*plantingStats
	sowingOffsets  []int
}

func newEvaluator(e *Engine, req EvaluationRequest) (*evaluator, error) {
	if req.Forecast != nil && len(req.Forecast) != planningHorizonMonths {
		return nil, fmt.Errorf("forecast must have %d months: %w", planningHorizonMonths, ErrInvalidRequest)
	}

	if req.FieldFreeOffset < 0 || req.FieldFreeOffset >= planningHorizonDays {
		return nil, fmt.Errorf("field free offset %d out of range: %w", req.FieldFreeOffset, ErrInvalidRequest)
	}

	ev := &evaluator{engine: e, req: req, cache: map[plantingKey]*plantingStats{}, sowingOffsets: sowingOffsets(req.Start, req.FieldFreeOffset)}
	ev.analogs = analogStarts(req.Series, req.Start)

	if len(ev.analogs) < minimumAnalogs {
		return nil, fmt.Errorf("%d analog years: %w", len(ev.analogs), ErrInsufficientHistory)
	}

	if c, ok := e.byID[req.CurrentCrop]; ok {
		ev.currentFamily = c.Family
	}

	ev.availableWater = req.Profile.Soil.AvailableWaterPerMeter()
	if math.IsNaN(float64(ev.availableWater)) || ev.availableWater <= 0 {
		ev.availableWater = defaultAvailableWaterMm
	}

	ev.prepareMonths()
	ev.prepareDaily()
	ev.prepareCanicula()

	return ev, nil
}

func analogStarts(s *SiteSeries, start time.Time) []int {
	var out []int

	first, last := s.DateAt(0).Year(), s.DateAt(s.Len()-1).Year()
	for y := first; y <= last; y++ {
		day := min(start.Day(), daysIn(y, start.Month()))
		idx := s.IndexOf(time.Date(y, start.Month(), day, 0, 0, 0, 0, time.UTC))

		if idx < 0 {
			continue
		}

		if idx+planningHorizonDays+maximumCycleWindowDays > s.Len() {
			break
		}

		out = append(out, idx)
	}

	return out
}

func daysIn(year int, m time.Month) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func (ev *evaluator) prepareMonths() {
	start := ev.req.Start
	totals := make([][]float64, len(ev.analogs))
	forecast := make([]*TercileProbabilities, planningHorizonMonths)

	annual := 0.0
	for _, v := range ev.req.Profile.MonthlyPrecipMedian {
		annual += v
	}

	for k := range planningHorizonMonths {
		from := offsetOf(start, start.AddDate(0, k, 0))
		to := offsetOf(start, start.AddDate(0, k+1, 0))
		ev.scoring.monthWindows[k] = [2]int{from, to}

		midMonth := start.AddDate(0, 0, (from+to)/2).Month()
		p50 := ev.req.Profile.MonthlyPrecipMedian[midMonth-1]
		ev.scoring.rainy[k] = p50 >= rainyMonthThresholdMm

		if annual > 0 {
			ev.scoring.erosivity[k] = p50 * p50 / annual
		}

		if ev.req.Forecast != nil && ev.req.Forecast[k] != nil {
			forecast[k] = ev.req.Forecast[k]
			ev.forecastMask[k] = true
			ev.forecastMonths++
		}
	}

	for a, idx := range ev.analogs {
		totals[a] = make([]float64, planningHorizonMonths)
		for k := range planningHorizonMonths {
			w := ev.scoring.monthWindows[k]
			totals[a][k] = sumPrecipitation(ev.req.Series, idx+w[0], idx+w[1])
		}
	}

	ev.monthWeights = AnalogMonthWeights(totals, forecast)
	ev.scoring.priority = ev.req.Priority
}

func offsetOf(start, t time.Time) int {
	return int(dateOnly(t).Sub(dateOnly(start)).Hours() / hoursPerDay)
}

func sumPrecipitation(s *SiteSeries, from, to int) float64 {
	sum := 0.0
	for i := from; i < to; i++ {
		sum += float64(s.Precipitation[i])
	}

	return sum
}

func (ev *evaluator) monthOf(offset int) int {
	for k := range planningHorizonMonths {
		if offset < ev.scoring.monthWindows[k][1] {
			return k
		}
	}

	return planningHorizonMonths - 1
}

func (ev *evaluator) prepareDaily() {
	s := ev.req.Series
	n := planningHorizonDays + maximumCycleWindowDays
	ev.meanTemp = make([]float64, n)
	ev.dailyBalance = make([]float64, planningHorizonDays)

	for o := range n {
		sum := 0.0
		for _, idx := range ev.analogs {
			sum += float64(s.TMean[idx+o])
		}

		ev.meanTemp[o] = sum / float64(len(ev.analogs))
	}

	for o := range planningHorizonDays {
		k := ev.monthOf(o)
		weighted, weight := 0.0, 0.0

		for a, idx := range ev.analogs {
			diff := float64(s.Precipitation[idx+o] - s.ET0[idx+o])
			if math.IsNaN(diff) {
				continue
			}

			weighted += ev.monthWeights[a][k] * diff
			weight += ev.monthWeights[a][k]
		}

		if weight > 0 {
			ev.dailyBalance[o] = weighted / weight
		}
	}
}

func (ev *evaluator) prepareCanicula() {
	s := ev.req.Series
	ev.canicula = map[int]DayWindow{}

	for y := s.DateAt(0).Year(); y <= s.DateAt(s.Len()-1).Year(); y++ {
		if w, ok := Canicula(s, y); ok {
			ev.canicula[y] = w
		}
	}
}

func (ev *evaluator) planting(c *Crop, sow int, mulch bool) *plantingStats {
	key := plantingKey{crop: c.ID, sow: sow, mulch: mulch}
	if st, ok := ev.cache[key]; ok {
		return st
	}

	st := ev.simulatePlanting(c, sow, mulch)
	ev.cache[key] = st

	return st
}

func (ev *evaluator) simulatePlanting(c *Crop, sow int, mulch bool) *plantingStats {
	nominal := c.CycleDays()
	scale := StageScaleForTemperature(Celsius(ev.meanTemperature(sow, nominal)), c.Temperature.Base)
	stages := c.ScaledStages(scale)
	cycle := stages[0] + stages[1] + stages[2] + stages[3]
	suitability, limit := Suitability(c, ev.req.Profile.Conditions(Celsius(ev.meanTemperature(sow, cycle)), ev.req.Irrigated))

	st := &plantingStats{
		crop: c, sow: sow, stages: stages, cycle: cycle, suitability: suitability, limit: limit,
		lossProbability: 1, months: ev.monthsCovered(sow, cycle),
	}

	if suitability > 0 && !ev.req.Irrigated && !ev.moistAtSowing(sow) {
		st.suitability, st.limit = 0, LimitDrySowing
	}

	if st.suitability <= 0 {
		return st
	}

	mulchFactor := 1.0
	if mulch {
		mulchFactor = mulchWaterMultiplier
	}

	ev.aggregate(st, ev.simulateAnalogs(c, stages, sow, mulchFactor))

	return st
}

func (ev *evaluator) moistAtSowing(sow int) bool {
	f := ev.req.Profile.Fortnights[FortnightOf(ev.req.Start.AddDate(0, 0, sow))]

	return f.PrecipP50 >= sowingMoistureRatio*f.ET0Mean
}

func (ev *evaluator) meanTemperature(sow, days int) float64 {
	end := min(sow+days, len(ev.meanTemp))
	sum := 0.0

	for o := sow; o < end; o++ {
		sum += ev.meanTemp[o]
	}

	return sum / float64(max(end-sow, 1))
}

func (ev *evaluator) monthsCovered(sow, cycle int) []int {
	first := ev.monthOf(sow)
	last := ev.monthOf(min(sow+cycle-1, planningHorizonDays-1))
	months := make([]int, 0, last-first+1)

	for k := first; k <= last; k++ {
		months = append(months, k)
	}

	return months
}

type analogOutcome struct {
	SeasonOutcome
	caniculaOverlap bool
}

func (ev *evaluator) simulateAnalogs(c *Crop, stages [4]int, sow int, mulchFactor float64) []analogOutcome {
	out := make([]analogOutcome, len(ev.analogs))

	for a, idx := range ev.analogs {
		o := SimulateSeason(SeasonInput{
			Series: ev.req.Series, SowIndex: idx + sow, Crop: c, Stages: stages,
			AvailableSoil: ev.availableWater, Irrigated: ev.req.Irrigated, MulchFactor: mulchFactor,
		})
		out[a] = analogOutcome{SeasonOutcome: o, caniculaOverlap: o.Complete && ev.floweringInCanicula(idx+sow, stages)}
	}

	return out
}

func (ev *evaluator) floweringInCanicula(sowIndex int, stages [4]int) bool {
	vegetativeEnd := stages[0] + stages[1]
	floweringDays := int(math.Ceil(float64(stages[2]) * floweringShareOfMidStage))
	from := ev.req.Series.DateAt(sowIndex + vegetativeEnd)
	to := from.AddDate(0, 0, floweringDays-1)

	w, ok := ev.canicula[from.Year()]
	if !ok || from.Year() != to.Year() {
		return false
	}

	return w.Overlaps(from.YearDay(), to.YearDay())
}

func (ev *evaluator) aggregate(st *plantingStats, outcomes []analogOutcome) {
	valid := make([]bool, len(outcomes))
	yields := make([]float64, len(outcomes))

	for a, o := range outcomes {
		valid[a] = o.Complete
		yields[a] = float64(o.RelativeYield)

		if o.Complete {
			st.analogsUsed++
		}
	}

	if st.analogsUsed*minimumAnalogShareForUse < minimumAnalogs {
		return
	}

	weights := PlantingWeights(ev.monthWeights, st.months, valid)
	st.lossProbability = 0

	for a, o := range outcomes {
		w := weights[a]
		if w == 0 {
			continue
		}

		if o.Failed() {
			st.lossProbability += w
		}

		st.expectedYield += w * float64(o.RelativeYield)
		st.irrigationMm += w * float64(o.IrrigationNeeded)
		st.heatDays += w * float64(o.HeatStressDays)

		if o.caniculaOverlap {
			st.caniculaShare += w
		}
	}

	st.medianYield = weightedMedian(yields, weights)
}
