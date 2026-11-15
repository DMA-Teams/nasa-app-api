package agro

import (
	"math"
	"slices"
)

type TercileProbabilities struct {
	Below  float64 `json:"below"`
	Normal float64 `json:"normal"`
	Above  float64 `json:"above"`
}

func (p TercileProbabilities) Normalized() TercileProbabilities {
	sum := p.Below + p.Normal + p.Above
	if sum <= 0 {
		return TercileProbabilities{Below: 1.0 / 3, Normal: 1.0 / 3, Above: 1.0 / 3}
	}

	return TercileProbabilities{Below: p.Below / sum, Normal: p.Normal / sum, Above: p.Above / sum}
}

func AnalogMonthWeights(totals [][]float64, forecast []*TercileProbabilities) [][]float64 {
	analogs := len(totals)
	weights := make([][]float64, analogs)

	for a := range weights {
		weights[a] = make([]float64, len(forecast))
	}

	for k := range forecast {
		column := monthWeights(totals, k, forecast[k])
		for a := range analogs {
			weights[a][k] = column[a]
		}
	}

	return weights
}

func monthWeights(totals [][]float64, k int, probs *TercileProbabilities) []float64 {
	out := make([]float64, len(totals))
	valid := make([]int, 0, len(totals))

	for a := range totals {
		if !math.IsNaN(totals[a][k]) {
			valid = append(valid, a)
		}
	}

	if len(valid) == 0 {
		return out
	}

	if probs == nil {
		for _, a := range valid {
			out[a] = 1 / float64(len(valid))
		}

		return out
	}

	slices.SortFunc(valid, func(x, y int) int {
		switch {
		case totals[x][k] < totals[y][k]:
			return -1
		case totals[x][k] > totals[y][k]:
			return 1
		default:
			return 0
		}
	})

	p := probs.Normalized()
	share := [3]float64{p.Below, p.Normal, p.Above}
	tercileOf := func(rank int) int { return min(rank*3/len(valid), 2) }
	counts := [3]int{}

	for rank := range valid {
		counts[tercileOf(rank)]++
	}

	for rank, a := range valid {
		t := tercileOf(rank)
		out[a] = share[t] / float64(counts[t])
	}

	return normalize(out)
}

func normalize(w []float64) []float64 {
	sum := 0.0
	for _, v := range w {
		sum += v
	}

	if sum <= 0 {
		return w
	}

	for i := range w {
		w[i] /= sum
	}

	return w
}

func PlantingWeights(monthWeights [][]float64, months []int, valid []bool) []float64 {
	out := make([]float64, len(monthWeights))

	for a := range monthWeights {
		if !valid[a] || len(months) == 0 {
			continue
		}

		sum := 0.0
		for _, k := range months {
			sum += monthWeights[a][k]
		}

		out[a] = sum / float64(len(months))
	}

	return normalize(out)
}
