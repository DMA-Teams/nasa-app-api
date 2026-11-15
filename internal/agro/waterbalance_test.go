package agro

import (
	"testing"
	"time"
)

func constantSeries(days int, rain, et0, tMax Millimeters) *SiteSeries {
	s := &SiteSeries{
		Start:         time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Precipitation: make([]Millimeters, days),
		ET0:           make([]Millimeters, days),
		TMean:         make([]Celsius, days),
		TMax:          make([]Celsius, days),
		TMin:          make([]Celsius, days),
	}

	for i := range days {
		s.Precipitation[i] = rain
		s.ET0[i] = et0
		s.TMax[i] = Celsius(tMax)
		s.TMean[i] = Celsius(tMax) - 6
		s.TMin[i] = Celsius(tMax) - 12
	}

	return s
}

func seasonFor(s *SiteSeries, irrigated bool) SeasonOutcome {
	c := testMaize()

	return SimulateSeason(SeasonInput{
		Series: s, SowIndex: 40, Crop: c, Stages: c.StageDays,
		AvailableSoil: 140, Irrigated: irrigated, MulchFactor: 1,
	})
}

func TestSimulateSeason_AmpleRainGivesFullYield(t *testing.T) {
	t.Parallel()

	out := seasonFor(constantSeries(400, 8, 4, 30), false)

	if !out.Complete || out.RelativeYield < 0.99 || out.StressDays != 0 {
		t.Errorf("ample rain: %+v, want complete, full yield, no stress", out)
	}
}

func TestSimulateSeason_DroughtFailsWithoutIrrigation(t *testing.T) {
	t.Parallel()

	out := seasonFor(constantSeries(400, 0, 5, 30), false)

	if !out.Failed() {
		t.Errorf("drought yield = %.2f, want < %.2f", out.RelativeYield, failedYieldThreshold)
	}

	for phase := range out.ETaByPhase {
		if out.ETaByPhase[phase] > out.ETcByPhase[phase] {
			t.Errorf("phase %d: ETa %.1f > ETc %.1f", phase, out.ETaByPhase[phase], out.ETcByPhase[phase])
		}
	}
}

func TestSimulateSeason_IrrigationRemovesStress(t *testing.T) {
	t.Parallel()

	out := seasonFor(constantSeries(400, 0, 5, 30), true)

	if out.RelativeYield < 0.95 || out.IrrigationNeeded <= 0 {
		t.Errorf("irrigated drought: yield %.2f irrigation %.1f, want >= 0.95 and > 0", out.RelativeYield, out.IrrigationNeeded)
	}
}

func TestSimulateSeason_HeatStressCountedInFlowering(t *testing.T) {
	t.Parallel()

	out := seasonFor(constantSeries(400, 8, 4, 38), false)
	if out.HeatStressDays == 0 {
		t.Error("heat stress days = 0, want > 0 when TMax exceeds the threshold")
	}
}

func TestSimulateSeason_IncompleteWhenSeriesTooShort(t *testing.T) {
	t.Parallel()

	if out := seasonFor(constantSeries(100, 8, 4, 30), false); out.Complete {
		t.Error("season beyond the series end must be incomplete")
	}
}

func TestRelativeYield_NeverNegative(t *testing.T) {
	t.Parallel()

	c := testMaize()
	got := RelativeYield(c, [4]Millimeters{}, [4]Millimeters{100, 100, 100, 100})

	if got < 0 || got > 1 {
		t.Errorf("RelativeYield = %.2f, want within [0, 1]", got)
	}
}
