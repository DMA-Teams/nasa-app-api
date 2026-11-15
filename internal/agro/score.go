package agro

import "math"

const (
	rainyMonthThresholdMm  = 100.0
	coveredMonthShare      = 0.5
	mulchCoverMultiplier   = 0.4
	mulchWaterMultiplier   = 1.1
	irrigationPenaltyPerMm = 1.0 / 40
	maxIrrigationPenalty   = 25.0
	nitrogenOffsetKgHa     = 60.0
	nitrogenRangeKgHa      = 180.0
	legumeDiversityBonus   = 0.3
	familyDiversityWeight  = 0.7
	scoreScale             = 100.0
	planningHorizonMonths  = 12
	planningHorizonDays    = 365
	bareSoilCoverFactor    = 1.0
	waterYieldShare        = 0.5
	waterSurvivalShare     = 0.5
)

type Priority int

const (
	PriorityUnknown Priority = iota
	PriorityBalanced
	PriorityWater
	PrioritySoil
)

func ParsePriority(s string) (Priority, bool) {
	switch s {
	case "", "balanced":
		return PriorityBalanced, true
	case "water":
		return PriorityWater, true
	case "soil":
		return PrioritySoil, true
	default:
		return PriorityUnknown, false
	}
}

func (p Priority) weights() (water, soil float64) {
	switch p {
	case PriorityWater:
		return 0.75, 0.25
	case PrioritySoil:
		return 0.30, 0.70
	case PriorityUnknown, PriorityBalanced:
		return 0.55, 0.45
	default:
		return 0.55, 0.45
	}
}

type Scores struct {
	Water float64 `json:"water"`
	Soil  float64 `json:"soil"`
	Total float64 `json:"total"`
}

type SoilIndicators struct {
	BareRainyMonths     int     `json:"bareRainyMonths"`
	RainyMonths         int     `json:"rainyMonths"`
	RelativeErosion     float64 `json:"relativeErosion"`
	NitrogenBalanceKgHa float64 `json:"nitrogenBalanceKgHa"`
	Families            int     `json:"families"`
	HasLegume           bool    `json:"hasLegume"`
	IrrigationMm        float64 `json:"irrigationMm"`
}

type scoringContext struct {
	monthWindows [planningHorizonMonths][2]int
	erosivity    [planningHorizonMonths]float64
	rainy        [planningHorizonMonths]bool
	priority     Priority
}

func scoreRotation(ctx *scoringContext, plantings []*plantingStats, mulch bool) (Scores, SoilIndicators) {
	if len(plantings) == 0 {
		return Scores{}, SoilIndicators{}
	}

	soil := soilIndicators(ctx, plantings, mulch)
	water := waterScore(plantings, soil.IrrigationMm)
	soilScore := soilScoreOf(soil, len(plantings))
	ww, ws := ctx.priority.weights()

	return Scores{
		Water: round1(water),
		Soil:  round1(soilScore),
		Total: round1(ww*water + ws*soilScore),
	}, soil
}

func waterScore(plantings []*plantingStats, irrigationMm float64) float64 {
	weighted, days := 0.0, 0.0

	for _, p := range plantings {
		weighted += float64(p.cycle) * (waterSurvivalShare*(1-p.lossProbability) + waterYieldShare*p.expectedYield)
		days += float64(p.cycle)
	}

	penalty := math.Min(maxIrrigationPenalty, irrigationMm*irrigationPenaltyPerMm)

	return Clamp(scoreScale*weighted/days-penalty, 0, scoreScale)
}

func soilIndicators(ctx *scoringContext, plantings []*plantingStats, mulch bool) SoilIndicators {
	var ind SoilIndicators

	families := map[string]bool{}
	erosionNumerator, erosionDenominator := 0.0, 0.0

	for _, p := range plantings {
		families[p.crop.Family] = true
		ind.HasLegume = ind.HasLegume || p.crop.IsLegume()
		ind.NitrogenBalanceKgHa += p.crop.NitrogenFixationKgHa*p.expectedYield - p.crop.NitrogenRemovalKgPerTon*p.crop.ReferenceYieldTonHa*p.expectedYield
		ind.IrrigationMm += p.irrigationMm
	}

	for k := range planningHorizonMonths {
		from, to := ctx.monthWindows[k][0], ctx.monthWindows[k][1]
		cover, covered := monthCover(plantings, from, to)

		if mulch {
			cover *= mulchCoverMultiplier
			covered = true
		}

		if ctx.rainy[k] {
			ind.RainyMonths++
			if !covered {
				ind.BareRainyMonths++
			}
		}

		erosionNumerator += ctx.erosivity[k] * cover
		erosionDenominator += ctx.erosivity[k]
	}

	if erosionDenominator > 0 {
		ind.RelativeErosion = round3(erosionNumerator / erosionDenominator)
	}

	ind.Families = len(families)
	ind.NitrogenBalanceKgHa = round1(ind.NitrogenBalanceKgHa)
	ind.IrrigationMm = round1(ind.IrrigationMm)

	return ind
}

func monthCover(plantings []*plantingStats, from, to int) (float64, bool) {
	best := bareSoilCoverFactor
	length := float64(max(to-from, 1))
	coveredDays := 0

	for day := from; day < to; day++ {
		for _, p := range plantings {
			if day >= p.sow && day < p.sow+p.cycle {
				coveredDays++

				break
			}
		}
	}

	mid := (from + to) / 2
	for _, p := range plantings {
		if mid >= p.sow && mid < p.sow+p.cycle {
			stage := StageOnDay(p.stages, mid-p.sow)
			best = math.Min(best, p.crop.CoverFactorByStage[stage])
		}
	}

	return best, float64(coveredDays)/length >= coveredMonthShare
}

func soilScoreOf(ind SoilIndicators, plantings int) float64 {
	bare := 1.0
	if ind.RainyMonths > 0 {
		bare = 1 - float64(ind.BareRainyMonths)/float64(ind.RainyMonths)
	}

	nitrogen := Clamp((ind.NitrogenBalanceKgHa+nitrogenOffsetKgHa)/nitrogenRangeKgHa, 0, 1)
	diversity := familyDiversityWeight * float64(ind.Families) / float64(max(plantings, 1))

	if ind.HasLegume {
		diversity += legumeDiversityBonus
	}

	return scoreScale * (bare + (1 - ind.RelativeErosion) + nitrogen + Clamp(diversity, 0, 1)) / 4
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
