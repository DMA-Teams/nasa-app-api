package agro

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

const (
	minimumAnalogs           = 10
	maximumCycleWindowDays   = 400
	candidateLossThreshold   = 0.4
	candidateSuitability     = 0.5
	relayMinimumDays         = 30
	maxPlantingsPerRotation  = 3
	beamWidth                = 150
	generatedRotations       = 3
	defaultAvailableWaterMm  = 120
	caniculaReasonShare      = 0.25
	heatReasonDays           = 1.0
	unsuitableReasonShare    = 0.5
	minimumAnalogShareForUse = 2
)

var (
	ErrInsufficientHistory = errors.New("not enough complete analog years in the climate series")
	ErrUnknownCrop         = errors.New("unknown crop")
	ErrInvalidRequest      = errors.New("invalid evaluation request")
)

type Template struct {
	ID    string
	Name  string
	Crops []string
	Relay []bool
	Mulch bool
}

type Engine struct {
	crops     []*Crop
	byID      map[string]*Crop
	templates []Template
}

func NewEngine(crops []*Crop, templates []Template) (*Engine, error) {
	byID := make(map[string]*Crop, len(crops))
	for _, c := range crops {
		byID[c.ID] = c
	}

	for _, t := range templates {
		if len(t.Relay) != len(t.Crops) {
			return nil, fmt.Errorf("template %s: relay flags do not match crops: %w", t.ID, ErrInvalidRequest)
		}

		for i, id := range t.Crops {
			c, ok := byID[id]
			if !ok {
				return nil, fmt.Errorf("template %s references %s: %w", t.ID, id, ErrUnknownCrop)
			}

			if t.Relay[i] && (i == 0 || !byID[t.Crops[i-1]].AcceptsRelay(c)) {
				return nil, fmt.Errorf("template %s: %s cannot be relayed: %w", t.ID, id, ErrInvalidRequest)
			}
		}
	}

	return &Engine{crops: crops, byID: byID, templates: templates}, nil
}

func (e *Engine) Crop(id string) (*Crop, bool) {
	c, ok := e.byID[id]

	return c, ok
}

func (e *Engine) Crops() []*Crop {
	return slices.Clone(e.crops)
}

type EvaluationRequest struct {
	Start           time.Time
	Series          *SiteSeries
	Profile         *SiteProfile
	Irrigated       bool
	Priority        Priority
	CurrentRotation []string
	CurrentCrop     string
	FieldFreeOffset int
	Forecast        []*TercileProbabilities
}

type RotationKind string

const (
	KindCurrent   RotationKind = "current"
	KindTemplate  RotationKind = "template"
	KindGenerated RotationKind = "generated"
)

type PlantingResult struct {
	CropID                string   `json:"cropId"`
	CropName              string   `json:"cropName"`
	Family                string   `json:"family"`
	Role                  string   `json:"role"`
	Season                string   `json:"season"`
	SowDate               string   `json:"sowDate"`
	HarvestDate           string   `json:"harvestDate"`
	CycleDays             int      `json:"cycleDays"`
	Relay                 bool     `json:"relay"`
	LossProbability       float64  `json:"lossProbability"`
	ExpectedRelativeYield float64  `json:"expectedRelativeYield"`
	MedianRelativeYield   float64  `json:"medianRelativeYield"`
	IrrigationMm          float64  `json:"irrigationMm"`
	HeatStressDays        float64  `json:"heatStressDays"`
	CaniculaFloweringRisk float64  `json:"caniculaFloweringRisk"`
	Suitability           float64  `json:"suitability"`
	LimitingFactor        string   `json:"limitingFactor,omitempty"`
	AnalogYears           int      `json:"analogYears"`
	LowConfidence         []string `json:"lowConfidenceParameters,omitempty"`
}

type RotationResult struct {
	ID        string           `json:"id"`
	Kind      RotationKind     `json:"kind"`
	Name      string           `json:"name"`
	Feasible  bool             `json:"feasible"`
	Mulch     bool             `json:"mulch"`
	Plantings []PlantingResult `json:"plantings"`
	Scores    Scores           `json:"scores"`
	Soil      SoilIndicators   `json:"soil"`
	Reasons   []Reason         `json:"reasons"`
	Quarters  []PeriodSummary  `json:"quarters"`
	Semesters []PeriodSummary  `json:"semesters"`
	Signature string           `json:"signature"`
}

type Evaluation struct {
	Start          string           `json:"start"`
	AnalogYears    int              `json:"analogYears"`
	ForecastMonths int              `json:"forecastMonths"`
	Rotations      []RotationResult `json:"rotations"`
	Best           string           `json:"best"`
	BestMargin     float64          `json:"bestMargin"`
}

func (e *Engine) Evaluate(ctx context.Context, req EvaluationRequest) (Evaluation, error) {
	if req.Series == nil || req.Profile == nil {
		return Evaluation{}, fmt.Errorf("series and profile are required: %w", ErrInvalidRequest)
	}

	ev, err := newEvaluator(e, req)
	if err != nil {
		return Evaluation{}, err
	}

	rotations := make([]RotationResult, 0, 2+len(e.templates)+generatedRotations)

	if len(req.CurrentRotation) > 0 {
		r, err := ev.sequenceRotation("current", KindCurrent, "Rotación actual", req.CurrentRotation, nil, false)
		if err != nil {
			return Evaluation{}, err
		}

		rotations = append(rotations, r)
	}

	for _, t := range e.templates {
		if err := ctx.Err(); err != nil {
			return Evaluation{}, fmt.Errorf("evaluate templates: %w", err)
		}

		r, err := ev.sequenceRotation(t.ID, KindTemplate, t.Name, t.Crops, t.Relay, t.Mulch)
		if err != nil {
			return Evaluation{}, err
		}

		rotations = append(rotations, r)
	}

	generated, err := ev.generate(ctx, signatures(rotations))
	if err != nil {
		return Evaluation{}, err
	}

	rotations = append(rotations, generated...)

	return ev.summarize(rotations), nil
}

func signatures(rotations []RotationResult) map[string]bool {
	set := make(map[string]bool, len(rotations))
	for _, r := range rotations {
		set[r.Signature] = true
	}

	return set
}

func (ev *evaluator) summarize(rotations []RotationResult) Evaluation {
	out := Evaluation{
		Start:          ev.req.Start.Format(time.DateOnly),
		AnalogYears:    len(ev.analogs),
		ForecastMonths: ev.forecastMonths,
		Rotations:      rotations,
	}

	best, second := -1.0, -1.0

	for _, r := range rotations {
		if !r.Feasible {
			continue
		}

		switch {
		case r.Scores.Total > best:
			second = best
			best = r.Scores.Total
			out.Best = r.ID
		case r.Scores.Total > second:
			second = r.Scores.Total
		}
	}

	if second >= 0 {
		out.BestMargin = round1(best - second)
	}

	return out
}

func (ev *evaluator) sequenceRotation(id string, kind RotationKind, name string, crops []string, relay []bool, mulch bool) (RotationResult, error) {
	if relay == nil {
		relay = make([]bool, len(crops))
	}

	plantings, relays, feasible, err := ev.evaluateSequence(crops, relay, mulch)
	if err != nil {
		return RotationResult{}, err
	}

	return ev.buildRotation(id, kind, name, plantings, relays, mulch, feasible), nil
}

func (ev *evaluator) evaluateSequence(ids []string, relay []bool, mulch bool) ([]*plantingStats, []bool, bool, error) {
	crops := make([]*Crop, len(ids))

	for i, id := range ids {
		c, ok := ev.engine.byID[id]
		if !ok {
			return nil, nil, false, fmt.Errorf("crop %q: %w", id, ErrUnknownCrop)
		}

		crops[i] = c
	}

	search := sequenceSearch{ev: ev, crops: crops, relay: relay, mulch: mulch}
	search.walk(nil, nil)

	if search.best.plantings == nil {
		return search.partial.plantings, search.partial.relay, false, nil
	}

	return search.best.plantings, search.best.relay, true, nil
}

type sequenceSearch struct {
	ev      *evaluator
	crops   []*Crop
	relay   []bool
	mulch   bool
	best    chain
	partial chain
}

func (s *sequenceSearch) walk(plantings []*plantingStats, relays []bool) {
	i := len(plantings)
	if i == len(s.crops) {
		s.keep(&s.best, plantings, relays)

		return
	}

	if i > len(s.partial.plantings) || (i == len(s.partial.plantings) && i > 0) {
		s.keep(&s.partial, plantings, relays)
	}

	for _, relay := range s.relayOptions(i) {
		for _, o := range s.ev.offsets() {
			st := s.ev.planting(s.crops[i], o, s.mulch)
			if st.suitability <= 0 || !s.ev.fits(plantings, st, relay, false) {
				continue
			}

			s.walk(append(slices.Clone(plantings), st), append(slices.Clone(relays), relay))
		}
	}
}

func (s *sequenceSearch) relayOptions(i int) []bool {
	switch {
	case i == 0 || !s.crops[i-1].AcceptsRelay(s.crops[i]):
		return []bool{false}
	case s.relay[i]:
		return []bool{true}
	default:
		return []bool{false, true}
	}
}

func (s *sequenceSearch) keep(target *chain, plantings []*plantingStats, relays []bool) {
	scores, _ := scoreRotation(&s.ev.scoring, plantings, s.mulch)
	if target.plantings == nil || len(plantings) > len(target.plantings) || scores.Total > target.total {
		*target = chain{plantings: plantings, relay: relays, total: scores.Total}
	}
}

func (ev *evaluator) offsets() []int {
	return ev.sowingOffsets
}

func sowingOffsets(start time.Time, from int) []int {
	out := make([]int, 0, fortnightsPerYear+1)

	for o := from; o < planningHorizonDays; o++ {
		if d := start.AddDate(0, 0, o).Day(); d == 1 || d == fortnightSplitDay+1 {
			out = append(out, o)
		}
	}

	return out
}

func (ev *evaluator) fits(chain []*plantingStats, next *plantingStats, relay, enforceFamilies bool) bool {
	if len(chain) == 0 {
		return next.sow >= ev.req.FieldFreeOffset && (!enforceFamilies || next.crop.Family != ev.currentFamily)
	}

	last := chain[len(chain)-1]
	if enforceFamilies && last.crop.Family == next.crop.Family {
		return false
	}

	if relay {
		if !last.crop.AcceptsRelay(next.crop) {
			return false
		}

		from, to := last.crop.RelayWindow(next.crop, last.stages)

		return next.sow >= last.sow+from && next.sow < last.sow+to
	}

	end := 0
	for _, p := range chain {
		end = max(end, p.sow+p.cycle)
	}

	return next.sow >= end
}

type chain struct {
	plantings []*plantingStats
	relay     []bool
	total     float64
}

func (ev *evaluator) generate(ctx context.Context, existing map[string]bool) ([]RotationResult, error) {
	candidates := ev.candidates()
	level := make([]chain, 0, len(candidates))

	for _, c := range candidates {
		if ev.fits(nil, c, false, true) {
			level = append(level, ev.scored([]*plantingStats{c}, []bool{false}))
		}
	}

	pool := slices.Clone(level)

	for depth := 2; depth <= maxPlantingsPerRotation; depth++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("generate rotations: %w", err)
		}

		level = ev.extend(topChains(level, beamWidth), candidates)
		pool = append(pool, level...)
	}

	slices.SortStableFunc(pool, func(a, b chain) int { return compareDesc(a.total, b.total) })

	out := make([]RotationResult, 0, generatedRotations)

	for _, ch := range pool {
		sig := chainSignature(ch.plantings)
		if existing[sig] || !hasHarvest(ch.plantings) {
			continue
		}

		existing[sig] = true
		id := fmt.Sprintf("generated-%d", len(out)+1)
		out = append(out, ev.buildRotation(id, KindGenerated, ev.chainName(ch.plantings), ch.plantings, ch.relay, false, true))

		if len(out) == generatedRotations {
			break
		}
	}

	return out, nil
}

func hasHarvest(plantings []*plantingStats) bool {
	return slices.ContainsFunc(plantings, func(p *plantingStats) bool { return p.crop.Role == RoleHarvest })
}

func (ev *evaluator) extend(level []chain, candidates []*plantingStats) []chain {
	next := make([]chain, 0, len(level)*4)

	for _, ch := range level {
		for _, c := range candidates {
			for _, relay := range []bool{false, true} {
				if !ev.fits(ch.plantings, c, relay, true) {
					continue
				}

				plantings := append(slices.Clone(ch.plantings), c)
				relays := append(slices.Clone(ch.relay), relay)
				next = append(next, ev.scored(plantings, relays))
			}
		}
	}

	return next
}

func (ev *evaluator) scored(plantings []*plantingStats, relay []bool) chain {
	scores, _ := scoreRotation(&ev.scoring, plantings, false)

	return chain{plantings: plantings, relay: relay, total: scores.Total}
}

func topChains(level []chain, n int) []chain {
	sorted := slices.Clone(level)
	slices.SortStableFunc(sorted, func(a, b chain) int { return compareDesc(a.total, b.total) })

	return sorted[:min(n, len(sorted))]
}

func compareDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}

func (ev *evaluator) candidates() []*plantingStats {
	var out []*plantingStats

	for _, c := range ev.engine.crops {
		for _, o := range ev.offsets() {
			st := ev.planting(c, o, false)
			if st.suitability >= candidateSuitability && st.lossProbability <= candidateLossThreshold && st.analogsUsed >= minimumAnalogs {
				out = append(out, st)
			}
		}
	}

	return out
}

func chainSignature(plantings []*plantingStats) string {
	ids := make([]string, len(plantings))
	for i, p := range plantings {
		ids[i] = p.crop.ID
	}

	return strings.Join(ids, ">")
}

func (ev *evaluator) chainName(plantings []*plantingStats) string {
	names := make([]string, len(plantings))
	for i, p := range plantings {
		names[i] = p.crop.Name
	}

	return strings.Join(names, " → ")
}

func (ev *evaluator) buildRotation(id string, kind RotationKind, name string, plantings []*plantingStats, relay []bool, mulch, feasible bool) RotationResult {
	scores, soil := scoreRotation(&ev.scoring, plantings, mulch)
	results := make([]PlantingResult, len(plantings))

	for i, p := range plantings {
		results[i] = ev.plantingResult(p, relay[i])
	}

	quarters, semesters := ev.periods(plantings)

	return RotationResult{
		ID: id, Kind: kind, Name: name, Feasible: feasible, Mulch: mulch,
		Plantings: results, Scores: scores, Soil: soil,
		Reasons:  ev.reasons(plantings, soil, feasible),
		Quarters: quarters, Semesters: semesters,
		Signature: chainSignature(plantings),
	}
}

func (ev *evaluator) plantingResult(p *plantingStats, relay bool) PlantingResult {
	sow := ev.req.Start.AddDate(0, 0, p.sow)

	return PlantingResult{
		CropID: p.crop.ID, CropName: p.crop.Name, Family: p.crop.Family, Role: roleName(p.crop.Role),
		Season: SeasonOf(sow.Month()), SowDate: sow.Format(time.DateOnly),
		HarvestDate: sow.AddDate(0, 0, p.cycle).Format(time.DateOnly), CycleDays: p.cycle, Relay: relay,
		LossProbability: round3(p.lossProbability), ExpectedRelativeYield: round3(p.expectedYield),
		MedianRelativeYield: round3(p.medianYield), IrrigationMm: round1(p.irrigationMm),
		HeatStressDays: round1(p.heatDays), CaniculaFloweringRisk: round3(p.caniculaShare),
		Suitability: round3(float64(p.suitability)), LimitingFactor: string(p.limit),
		AnalogYears: p.analogsUsed, LowConfidence: p.crop.LowConfidenceFields,
	}
}

func roleName(r CropRole) string {
	if r == RoleCover {
		return "cover"
	}

	return "harvest"
}

func SeasonOf(m time.Month) string {
	switch m {
	case time.May, time.June, time.July:
		return "primera"
	case time.August, time.September, time.October:
		return "postrera"
	case time.November, time.December, time.January:
		return "apante"
	case time.February, time.March, time.April:
		return "verano"
	default:
		return "verano"
	}
}

func weightedMedian(values, weights []float64) float64 {
	idx := make([]int, 0, len(values))

	for i := range values {
		if weights[i] > 0 {
			idx = append(idx, i)
		}
	}

	if len(idx) == 0 {
		return math.NaN()
	}

	slices.SortFunc(idx, func(a, b int) int { return compareDesc(values[b], values[a]) })

	cumulative := 0.0
	for _, i := range idx {
		cumulative += weights[i]
		if cumulative >= 0.5 {
			return values[i]
		}
	}

	return values[idx[len(idx)-1]]
}
