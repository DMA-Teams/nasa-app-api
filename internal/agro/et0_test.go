package agro

import (
	"math"
	"testing"
)

func assertNear(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()

	if math.Abs(got-want) > tolerance {
		t.Errorf("%s = %.4f, want %.4f ± %.4f", name, got, want, tolerance)
	}
}

func TestExtraterrestrialRadiation_FAO56Example8(t *testing.T) {
	t.Parallel()

	ra := ExtraterrestrialRadiation(-20, 246)
	assertNear(t, "Ra", float64(ra), 32.2, 0.1)
}

func TestPenmanMonteith_FAO56Example18(t *testing.T) {
	t.Parallel()

	tMax, tMin := Celsius(21.5), Celsius(12.3)
	loc := Location{LatitudeDeg: 50.8, Elevation: 100}
	meteo := DailyMeteo{
		TMax:                tMax,
		TMin:                tMin,
		ActualVaporPressure: VaporPressureFromHumidityExtremes(tMax, tMin, 84, 63),
		WindAt2m:            2.078,
		SolarRadiation:      22.07,
	}

	assertNear(t, "ea", float64(meteo.ActualVaporPressure), 1.409, 0.01)
	assertNear(t, "Ra", float64(ExtraterrestrialRadiation(loc.LatitudeDeg, 187)), 41.09, 0.1)
	assertNear(t, "Rn", float64(NetRadiation(meteo, ExtraterrestrialRadiation(loc.LatitudeDeg, 187), loc.Elevation)), 13.28, 0.1)
	assertNear(t, "ET0", float64(PenmanMonteith(meteo, loc, 187)), 3.9, 0.1)
}

func TestHargreaves_EquationCheckOnFAO56Example18Inputs(t *testing.T) {
	t.Parallel()

	ra := ExtraterrestrialRadiation(50.8, 187)
	assertNear(t, "ET0", float64(Hargreaves(21.5, 12.3, ra)), 4.06, 0.03)
}

func TestHargreaves_NeverNegative(t *testing.T) {
	t.Parallel()

	if got := Hargreaves(-30, -10, 5); got < 0 {
		t.Errorf("Hargreaves = %v, want >= 0", got)
	}
}

func TestAtmosphericPressure_FAO56Example2(t *testing.T) {
	t.Parallel()

	assertNear(t, "P", float64(AtmosphericPressure(1800)), 81.8, 0.1)
}

func TestEnergyToEvaporation(t *testing.T) {
	t.Parallel()

	assertNear(t, "mm", float64(EnergyToEvaporation(4.15)), 1.694, 0.001)
}
