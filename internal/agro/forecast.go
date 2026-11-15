package agro

import (
	"math"
	"time"
)

const (
	tercileHalfWidthInStdDevs = 0.43
	minimumEnsembleMembers    = 10
)

type MonthlyTerciles struct {
	Year          int                  `json:"year"`
	Month         time.Month           `json:"month"`
	Probabilities TercileProbabilities `json:"probabilities"`
}

type CellForecast struct {
	Issued string            `json:"issued"`
	Model  string            `json:"model"`
	Months []MonthlyTerciles `json:"months"`
}

func (f CellForecast) For(year int, month time.Month) (TercileProbabilities, bool) {
	for _, m := range f.Months {
		if m.Year == year && m.Month == month {
			return m.Probabilities, true
		}
	}

	return TercileProbabilities{}, false
}

func TercilesFromEnsemble(memberAnomalies []float64, climatologicalStdDev float64) (TercileProbabilities, bool) {
	if len(memberAnomalies) < minimumEnsembleMembers || climatologicalStdDev <= 0 || math.IsNaN(climatologicalStdDev) {
		return TercileProbabilities{}, false
	}

	threshold := tercileHalfWidthInStdDevs * climatologicalStdDev

	var below, above, valid float64

	for _, a := range memberAnomalies {
		if math.IsNaN(a) {
			continue
		}

		valid++

		switch {
		case a < -threshold:
			below++
		case a > threshold:
			above++
		}
	}

	if valid < minimumEnsembleMembers {
		return TercileProbabilities{}, false
	}

	return TercileProbabilities{Below: below / valid, Normal: (valid - below - above) / valid, Above: above / valid}, true
}

func StdDev(values []float64) float64 {
	m := mean(values)
	sum, n := 0.0, 0

	for _, v := range values {
		if !math.IsNaN(v) {
			sum += (v - m) * (v - m)
			n++
		}
	}

	if n < 2 {
		return math.NaN()
	}

	return math.Sqrt(sum / float64(n-1))
}

func BlendForecasts(forecasts []CellForecast, weights []float64, start time.Time) []*TercileProbabilities {
	out := make([]*TercileProbabilities, planningHorizonMonths)

	for k := range planningHorizonMonths {
		from := start.AddDate(0, k, 0)
		mid := from.AddDate(0, 0, offsetOf(from, start.AddDate(0, k+1, 0))/2)

		var blended TercileProbabilities

		total := 0.0

		for i, f := range forecasts {
			p, ok := f.For(mid.Year(), mid.Month())
			if !ok {
				continue
			}

			w := weights[i]
			blended.Below += w * p.Below
			blended.Normal += w * p.Normal
			blended.Above += w * p.Above
			total += w
		}

		if total > 0 {
			b := blended.Normalized()
			out[k] = &b
		}
	}

	return out
}
