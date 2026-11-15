package agro

import "testing"

func TestSaxtonRawls_EquationCheckLoam(t *testing.T) {
	t.Parallel()

	w := SaxtonRawls(SoilTexture{Sand: 0.40, Clay: 0.20, OrganicMatterPercent: 2.5})

	assertNear(t, "FC", float64(w.FieldCapacity), 0.2796, 0.002)
	assertNear(t, "WP", float64(w.WiltingPoint), 0.1370, 0.002)
	assertNear(t, "AW", float64(w.AvailableWaterPerMeter()), 142.6, 2)
}

func TestSaxtonRawls_FieldCapacityAboveWiltingPoint(t *testing.T) {
	t.Parallel()

	for sand := 0.05; sand <= 0.9; sand += 0.05 {
		for clay := 0.05; clay+sand <= 0.95; clay += 0.05 {
			w := SaxtonRawls(SoilTexture{Sand: Fraction(sand), Clay: Fraction(clay), OrganicMatterPercent: 3})
			if w.FieldCapacity <= w.WiltingPoint {
				t.Fatalf("sand=%.2f clay=%.2f: FC %.3f <= WP %.3f", sand, clay, w.FieldCapacity, w.WiltingPoint)
			}
		}
	}
}

func TestOrganicMatterPercentFromCarbon(t *testing.T) {
	t.Parallel()

	assertNear(t, "OM", OrganicMatterPercentFromCarbon(55.7), 9.6, 0.01)
}
