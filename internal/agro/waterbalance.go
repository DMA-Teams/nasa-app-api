package agro

import "math"

const (
	preSeasonDays          = 30
	bareSoilEvaporationKc  = 0.3
	failedYieldThreshold   = 0.5
	depletionAdjustmentRef = 5.0
)

type SeasonInput struct {
	Series        *SiteSeries
	SowIndex      int
	Crop          *Crop
	Stages        [4]int
	AvailableSoil Millimeters
	Irrigated     bool
	MulchFactor   float64
}

type SeasonOutcome struct {
	Complete         bool
	RelativeYield    Fraction
	ETcByPhase       [4]Millimeters
	ETaByPhase       [4]Millimeters
	StressDays       int
	HeatStressDays   int
	IrrigationNeeded Millimeters
}

func (o SeasonOutcome) Failed() bool {
	return o.RelativeYield < failedYieldThreshold
}

func SimulateSeason(in SeasonInput) SeasonOutcome {
	cycle := in.Stages[0] + in.Stages[1] + in.Stages[2] + in.Stages[3]
	end := in.SowIndex + cycle

	if in.SowIndex < 0 || end > in.Series.Len() {
		return SeasonOutcome{}
	}

	taw := float64(in.AvailableSoil) * float64(in.Crop.MaxRootDepth) * math.Max(in.MulchFactor, 1)
	depletion, ok := preSeasonDepletion(in.Series, in.SowIndex, taw)

	if !ok {
		return SeasonOutcome{}
	}

	out := SeasonOutcome{Complete: true}

	for d := range cycle {
		i := in.SowIndex + d
		p := float64(in.Series.Precipitation[i])
		et0 := float64(in.Series.ET0[i])

		if math.IsNaN(p) || math.IsNaN(et0) {
			return SeasonOutcome{}
		}

		kc := in.Crop.KcOnDay(in.Stages, d)
		etc := kc * et0
		readily := adjustedDepletionFraction(float64(in.Crop.DepletionFraction), etc) * taw

		if in.Irrigated && depletion > readily {
			out.IrrigationNeeded += Millimeters(depletion)
			depletion = 0
		}

		ks := waterStressCoefficient(depletion, readily, taw)
		eta := ks * etc
		phase := PhaseOnDay(in.Stages, d)
		out.ETcByPhase[phase] += Millimeters(etc)
		out.ETaByPhase[phase] += Millimeters(eta)

		if ks < 1 {
			out.StressDays++
		}

		if phase == PhaseFlowering && in.Series.TMax[i] > in.Crop.HeatStressTMax {
			out.HeatStressDays++
		}

		depletion = Clamp(depletion-p+eta, 0, taw)
	}

	out.RelativeYield = RelativeYield(in.Crop, out.ETaByPhase, out.ETcByPhase)

	return out
}

func preSeasonDepletion(s *SiteSeries, sowIndex int, taw float64) (float64, bool) {
	start := sowIndex - preSeasonDays
	if start < 0 {
		return taw / 2, true
	}

	depletion := taw / 2

	for i := start; i < sowIndex; i++ {
		p, et0 := float64(s.Precipitation[i]), float64(s.ET0[i])
		if math.IsNaN(p) || math.IsNaN(et0) {
			return 0, false
		}

		depletion = Clamp(depletion-p+bareSoilEvaporationKc*et0, 0, taw)
	}

	return depletion, true
}

func adjustedDepletionFraction(p, etc float64) float64 {
	return Clamp(p+0.04*(depletionAdjustmentRef-etc), 0.1, 0.8)
}

func waterStressCoefficient(depletion, readily, taw float64) float64 {
	if depletion <= readily || taw <= readily {
		return 1
	}

	return Clamp((taw-depletion)/(taw-readily), 0, 1)
}

func RelativeYield(c *Crop, eta, etc [4]Millimeters) Fraction {
	if c.HasKyByPhase {
		ratio := 1.0

		for i := range eta {
			if etc[i] <= 0 {
				continue
			}

			ratio *= 1 - c.KyByPhase[i]*(1-float64(eta[i])/float64(etc[i]))
		}

		return Fraction(Clamp(ratio, 0, 1))
	}

	var sumETa, sumETc Millimeters
	for i := range eta {
		sumETa += eta[i]
		sumETc += etc[i]
	}

	if sumETc <= 0 {
		return 1
	}

	return Fraction(Clamp(1-c.KyTotal*(1-float64(sumETa)/float64(sumETc)), 0, 1))
}
