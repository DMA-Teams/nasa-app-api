package agro

type Millimeters float64

type Celsius float64

type MegajoulesPerSquareMeterDay float64

type Kilopascals float64

type MetersPerSecond float64

type Meters float64

type Fraction float64

const LatentHeatOfVaporization = 2.45

func EnergyToEvaporation(e MegajoulesPerSquareMeterDay) Millimeters {
	return Millimeters(float64(e) / LatentHeatOfVaporization)
}

func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}
