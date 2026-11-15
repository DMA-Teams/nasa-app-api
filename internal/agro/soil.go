package agro

const organicCarbonToOrganicMatter = 1.724

type SoilTexture struct {
	Sand                 Fraction
	Clay                 Fraction
	OrganicMatterPercent float64
}

type SoilWater struct {
	FieldCapacity Fraction
	WiltingPoint  Fraction
}

func OrganicMatterPercentFromCarbon(organicCarbonGramsPerKg float64) float64 {
	return organicCarbonGramsPerKg / 10 * organicCarbonToOrganicMatter
}

func SaxtonRawls(t SoilTexture) SoilWater {
	s, c, om := float64(t.Sand), float64(t.Clay), Clamp(t.OrganicMatterPercent, 0, 8)

	wp1500t := -0.024*s + 0.487*c + 0.006*om + 0.005*s*om - 0.013*c*om + 0.068*s*c + 0.031
	wp := wp1500t + (0.14*wp1500t - 0.02)

	fc33t := -0.251*s + 0.195*c + 0.011*om + 0.006*s*om - 0.027*c*om + 0.452*s*c + 0.299
	fc := fc33t + (1.283*fc33t*fc33t - 0.374*fc33t - 0.015)

	wp = Clamp(wp, 0.01, 0.6)
	fc = Clamp(fc, wp+0.01, 0.7)

	return SoilWater{FieldCapacity: Fraction(fc), WiltingPoint: Fraction(wp)}
}

func (w SoilWater) AvailableWaterPerMeter() Millimeters {
	return Millimeters(1000 * float64(w.FieldCapacity-w.WiltingPoint))
}
