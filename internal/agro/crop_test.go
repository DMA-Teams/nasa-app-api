package agro

import "testing"

func testMaize() *Crop {
	return &Crop{
		ID: "maize", Name: "Maíz", Family: "Poaceae", Role: RoleHarvest,
		StageDays: [4]int{20, 35, 40, 30}, KcIni: 0.3, KcMid: 1.2, KcEnd: 0.35,
		MaxRootDepth: 1.0, DepletionFraction: 0.55, KyTotal: 1.25,
		KyByPhase: [4]float64{0.4, 1.5, 0.5, 0.2}, HasKyByPhase: true,
		Temperature:    TemperatureRange{Base: 10, AbsoluteLo: 10, OptimalLo: 18, OptimalHi: 32, AbsoluteHi: 40},
		HeatStressTMax: 35, PH: PHRange{AbsoluteLo: 4.5, OptimalLo: 5.5, OptimalHi: 7.5, AbsoluteHi: 8.5},
		AltitudeMin: 0, AltitudeMax: 2200, Irrigation: IrrigationOptional, Waterlogging: WaterloggingModerate,
		NitrogenRemovalKgPerTon: 16, ReferenceYieldTonHa: 1.8,
		CoverFactorByStage: [4]float64{0.8, 0.5, 0.25, 0.35},
		RelayHosts:         []RelayKind{RelayCover, RelayGrain},
	}
}

func TestKcOnDay_FollowsFAO56Curve(t *testing.T) {
	t.Parallel()

	c := testMaize()
	stages := c.StageDays

	tests := []struct {
		day  int
		want float64
	}{
		{0, 0.3},
		{19, 0.3},
		{54, 1.2},
		{60, 1.2},
		{124, 0.35},
	}

	for _, tt := range tests {
		assertNear(t, "Kc", c.KcOnDay(stages, tt.day), tt.want, 1e-9)
	}

	mid := c.KcOnDay(stages, 37)
	if mid <= 0.3 || mid >= 1.2 {
		t.Errorf("Kc during development = %.3f, want between 0.3 and 1.2", mid)
	}
}

func TestPhaseOnDay(t *testing.T) {
	t.Parallel()

	stages := [4]int{20, 35, 40, 30}

	tests := []struct {
		day  int
		want Phase
	}{
		{0, PhaseVegetative},
		{54, PhaseVegetative},
		{55, PhaseFlowering},
		{70, PhaseFlowering},
		{71, PhaseYieldFormation},
		{95, PhaseRipening},
	}

	for _, tt := range tests {
		if got := PhaseOnDay(stages, tt.day); got != tt.want {
			t.Errorf("PhaseOnDay(%d) = %d, want %d", tt.day, got, tt.want)
		}
	}
}

func TestStageScaleForTemperature(t *testing.T) {
	t.Parallel()

	assertNear(t, "reference", StageScaleForTemperature(24, 10), 1, 1e-9)

	if cold := StageScaleForTemperature(17, 10); cold <= 1 {
		t.Errorf("cold scale = %.2f, want > 1", cold)
	}

	if hot := StageScaleForTemperature(28, 10); hot >= 1 {
		t.Errorf("hot scale = %.2f, want < 1", hot)
	}

	assertNear(t, "below base", StageScaleForTemperature(5, 10), maximumStageScale, 1e-9)
}
