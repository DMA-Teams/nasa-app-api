package agro

import (
	"math"
	"slices"
	"time"
)

const (
	onsetSearchFromDOY    = 91
	onsetSearchToDOY      = 243
	onsetRainWindowDays   = 3
	onsetRainThresholdMm  = 20
	onsetCheckDays        = 30
	onsetMaxDrySpellDays  = 7
	dryDayEvaporationRate = 0.5
	caniculaFromDOY       = 166
	caniculaToDOY         = 243
	caniculaWindowDays    = 10
	caniculaDeficitRatio  = 0.5
	caniculaMinimumDays   = 5
	fortnightsPerYear     = 24
	fortnightSplitDay     = 15
)

type DayWindow struct {
	StartDOY int `json:"startDayOfYear"`
	EndDOY   int `json:"endDayOfYear"`
}

func (w DayWindow) Length() int {
	return w.EndDOY - w.StartDOY + 1
}

func (w DayWindow) Overlaps(startDOY, endDOY int) bool {
	return startDOY <= w.EndDOY && endDOY >= w.StartDOY
}

func Percentile(values []float64, p float64) float64 {
	clean := make([]float64, 0, len(values))

	for _, v := range values {
		if !math.IsNaN(v) {
			clean = append(clean, v)
		}
	}

	if len(clean) == 0 {
		return math.NaN()
	}

	slices.Sort(clean)
	rank := Clamp(p, 0, 1) * float64(len(clean)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))

	return clean[lo] + (clean[hi]-clean[lo])*(rank-float64(lo))
}

func yearStartIndex(s *SiteSeries, year int) int {
	return s.IndexOf(time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC))
}

func isDryDay(p, et0 Millimeters) bool {
	return float64(p) < dryDayEvaporationRate*float64(et0)
}

func OnsetOfRains(s *SiteSeries, year int) (int, bool) {
	base := yearStartIndex(s, year)

	for doy := onsetSearchFromDOY; doy <= onsetSearchToDOY; doy++ {
		i := base + doy - 1
		if i < 0 || i+onsetCheckDays+onsetRainWindowDays >= s.Len() {
			return 0, false
		}

		rain := Millimeters(0)
		for k := range onsetRainWindowDays {
			rain += s.Precipitation[i+k]
		}

		if math.IsNaN(float64(rain)) || rain < onsetRainThresholdMm {
			continue
		}

		if longestDrySpell(s, i+onsetRainWindowDays, onsetCheckDays) <= onsetMaxDrySpellDays {
			return doy, true
		}
	}

	return 0, false
}

func longestDrySpell(s *SiteSeries, from, days int) int {
	longest, run := 0, 0

	for i := from; i < from+days && i < s.Len(); i++ {
		if isDryDay(s.Precipitation[i], s.ET0[i]) {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}

	return longest
}

type CaniculaRule struct {
	DeficitRatio float64
	MinimumDays  int
}

var DefaultCaniculaRule = CaniculaRule{DeficitRatio: caniculaDeficitRatio, MinimumDays: caniculaMinimumDays}

func Canicula(s *SiteSeries, year int) (DayWindow, bool) {
	return DefaultCaniculaRule.Detect(s, year)
}

func (r CaniculaRule) Detect(s *SiteSeries, year int) (DayWindow, bool) {
	base := yearStartIndex(s, year)
	best, bestLen := DayWindow{}, 0
	runStart, runLen := 0, 0

	for doy := caniculaFromDOY; doy <= caniculaToDOY; doy++ {
		i := base + doy - 1
		if i < 0 || i+caniculaWindowDays > s.Len() {
			return DayWindow{}, false
		}

		if deficitWindow(s, i, r.DeficitRatio) {
			if runLen == 0 {
				runStart = doy
			}

			runLen++
			if runLen > bestLen {
				bestLen = runLen
				best = DayWindow{StartDOY: runStart, EndDOY: doy + caniculaWindowDays - 1}
			}
		} else {
			runLen = 0
		}
	}

	return best, bestLen >= r.MinimumDays
}

func deficitWindow(s *SiteSeries, i int, ratio float64) bool {
	var p, e Millimeters
	for k := range caniculaWindowDays {
		p += s.Precipitation[i+k]
		e += s.ET0[i+k]
	}

	return !math.IsNaN(float64(p)) && float64(p) < ratio*float64(e)
}

type FortnightClimate struct {
	Index         int     `json:"index"`
	Month         int     `json:"month"`
	Half          int     `json:"half"`
	PrecipP20     float64 `json:"precipitationP20Mm"`
	PrecipP50     float64 `json:"precipitationP50Mm"`
	ET0Mean       float64 `json:"et0MeanMm"`
	DeficitMedian float64 `json:"deficitMedianMm"`
}

func FortnightOf(t time.Time) int {
	half := 0
	if t.Day() > fortnightSplitDay {
		half = 1
	}

	return (int(t.Month())-1)*2 + half
}

func FortnightClimatology(s *SiteSeries) [fortnightsPerYear]FortnightClimate {
	type key struct{ year, fortnight int }

	rain := map[key]float64{}
	et := map[key]float64{}

	for i := range s.Len() {
		d := s.DateAt(i)
		k := key{d.Year(), FortnightOf(d)}
		rain[k] += float64(s.Precipitation[i])
		et[k] += float64(s.ET0[i])
	}

	var out [fortnightsPerYear]FortnightClimate

	for f := range fortnightsPerYear {
		var rains, ets []float64

		for k, v := range rain {
			if k.fortnight == f {
				rains = append(rains, v)
				ets = append(ets, et[k])
			}
		}

		p50 := Percentile(rains, 0.5)
		etMean := mean(ets)
		out[f] = FortnightClimate{
			Index: f, Month: f/2 + 1, Half: f%2 + 1,
			PrecipP20: Percentile(rains, 0.2), PrecipP50: p50,
			ET0Mean: etMean, DeficitMedian: p50 - etMean,
		}
	}

	return out
}

func mean(values []float64) float64 {
	sum, n := 0.0, 0

	for _, v := range values {
		if !math.IsNaN(v) {
			sum += v
			n++
		}
	}

	if n == 0 {
		return math.NaN()
	}

	return sum / float64(n)
}

func MonthlyMedianPrecipitation(s *SiteSeries) [12]float64 {
	totals := map[YearMonth]float64{}
	for i := range s.Len() {
		d := s.DateAt(i)
		totals[YearMonth{d.Year(), d.Month()}] += float64(s.Precipitation[i])
	}

	var out [12]float64

	for m := range 12 {
		var vals []float64

		for k, v := range totals {
			if int(k.Month) == m+1 {
				vals = append(vals, v)
			}
		}

		out[m] = Percentile(vals, 0.5)
	}

	return out
}
