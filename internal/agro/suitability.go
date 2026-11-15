package agro

const (
	altitudeToleranceMeters      = 200.0
	heavyClayPercent             = 45.0
	waterloggingSensitivePenalty = 0.6
	waterloggingModeratePenalty  = 0.85
)

type LimitingFactor string

const (
	LimitNone         LimitingFactor = ""
	LimitTemperature  LimitingFactor = "temperature"
	LimitPH           LimitingFactor = "ph"
	LimitAltitude     LimitingFactor = "altitude"
	LimitIrrigation   LimitingFactor = "irrigation"
	LimitWaterlogging LimitingFactor = "waterlogging"
	LimitDrySowing    LimitingFactor = "dry_sowing"
)

const sowingMoistureRatio = 0.5

type SiteConditions struct {
	MeanTemperature Celsius
	PH              float64
	ClayPercent     float64
	Elevation       Meters
	Irrigated       bool
}

func Trapezoid(v, absoluteLo, optimalLo, optimalHi, absoluteHi float64) float64 {
	switch {
	case v <= absoluteLo || v >= absoluteHi:
		return 0
	case v < optimalLo:
		return (v - absoluteLo) / (optimalLo - absoluteLo)
	case v > optimalHi:
		return (absoluteHi - v) / (absoluteHi - optimalHi)
	default:
		return 1
	}
}

func Suitability(c *Crop, s SiteConditions) (Fraction, LimitingFactor) {
	type factor struct {
		value float64
		name  LimitingFactor
	}

	t := c.Temperature
	factors := []factor{
		{Trapezoid(float64(s.MeanTemperature), float64(t.AbsoluteLo), float64(t.OptimalLo), float64(t.OptimalHi), float64(t.AbsoluteHi)), LimitTemperature},
		{Trapezoid(s.PH, c.PH.AbsoluteLo, c.PH.OptimalLo, c.PH.OptimalHi, c.PH.AbsoluteHi), LimitPH},
		{altitudeFactor(c, s.Elevation), LimitAltitude},
		{irrigationFactor(c, s.Irrigated), LimitIrrigation},
		{waterloggingFactor(c, s.ClayPercent), LimitWaterlogging},
	}

	best := factor{value: 1, name: LimitNone}
	for _, f := range factors {
		if f.value < best.value {
			best = f
		}
	}

	return Fraction(best.value), best.name
}

func altitudeFactor(c *Crop, z Meters) float64 {
	lo, hi, v := float64(c.AltitudeMin), float64(c.AltitudeMax), float64(z)

	return Trapezoid(v, lo-altitudeToleranceMeters, lo, hi, hi+altitudeToleranceMeters)
}

func irrigationFactor(c *Crop, irrigated bool) float64 {
	if c.Irrigation == IrrigationRequired && !irrigated {
		return 0
	}

	return 1
}

func waterloggingFactor(c *Crop, clayPercent float64) float64 {
	if clayPercent < heavyClayPercent {
		return 1
	}

	switch c.Waterlogging {
	case WaterloggingSensitive:
		return waterloggingSensitivePenalty
	case WaterloggingModerate:
		return waterloggingModeratePenalty
	case WaterloggingUnknown, WaterloggingTolerant:
		return 1
	default:
		return 1
	}
}
