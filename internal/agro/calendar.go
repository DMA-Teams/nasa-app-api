package agro

import "time"

const (
	quartersPerYear   = 4
	semestersPerYear  = 2
	monthsPerQuarter  = 3
	monthsPerSemester = 6
)

type PeriodSummary struct {
	Index                  int      `json:"index"`
	From                   string   `json:"from"`
	To                     string   `json:"to"`
	Sowing                 []string `json:"sowing"`
	InField                []string `json:"inField"`
	Harvest                []string `json:"harvest"`
	SoilCovered            bool     `json:"soilCovered"`
	ExpectedWaterBalanceMm float64  `json:"expectedWaterBalanceMm"`
	ForecastInfluenced     bool     `json:"forecastInfluenced"`
}

func (ev *evaluator) periods(plantings []*plantingStats) ([]PeriodSummary, []PeriodSummary) {
	return ev.periodsOf(plantings, quartersPerYear, monthsPerQuarter), ev.periodsOf(plantings, semestersPerYear, monthsPerSemester)
}

func (ev *evaluator) periodsOf(plantings []*plantingStats, count, months int) []PeriodSummary {
	out := make([]PeriodSummary, count)
	start := ev.req.Start

	for i := range count {
		fromDate := start.AddDate(0, i*months, 0)
		toDate := start.AddDate(0, (i+1)*months, 0)
		from, to := offsetOf(start, fromDate), min(offsetOf(start, toDate), planningHorizonDays)

		p := PeriodSummary{
			Index: i + 1, From: fromDate.Format(time.DateOnly), To: toDate.AddDate(0, 0, -1).Format(time.DateOnly),
			Sowing: []string{}, InField: []string{}, Harvest: []string{},
		}

		coveredDays := 0

		for day := from; day < to; day++ {
			p.ExpectedWaterBalanceMm += ev.dailyBalance[day]

			if inField(plantings, day) {
				coveredDays++
			}
		}

		for _, pl := range plantings {
			end := pl.sow + pl.cycle

			switch {
			case pl.sow >= from && pl.sow < to:
				p.Sowing = append(p.Sowing, pl.crop.ID)
			case pl.sow < from && end > from:
				p.InField = append(p.InField, pl.crop.ID)
			}

			if end >= from && end < to {
				p.Harvest = append(p.Harvest, pl.crop.ID)
			}
		}

		for k := i * months; k < (i+1)*months && k < planningHorizonMonths; k++ {
			p.ForecastInfluenced = p.ForecastInfluenced || ev.forecastMask[k]
		}

		p.SoilCovered = to > from && float64(coveredDays)/float64(to-from) >= coveredMonthShare
		p.ExpectedWaterBalanceMm = round1(p.ExpectedWaterBalanceMm)
		out[i] = p
	}

	return out
}

func inField(plantings []*plantingStats, day int) bool {
	for _, p := range plantings {
		if day >= p.sow && day < p.sow+p.cycle {
			return true
		}
	}

	return false
}
