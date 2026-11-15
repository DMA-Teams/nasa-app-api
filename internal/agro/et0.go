package agro

import "math"

const (
	solarConstant          = 0.0820
	stefanBoltzmann        = 4.903e-9
	referenceAlbedo        = 0.23
	daysPerYear            = 365.0
	kelvinOffset           = 273.16
	megajoulesToMillimeter = 0.408
)

type DailyMeteo struct {
	TMax                Celsius
	TMin                Celsius
	ActualVaporPressure Kilopascals
	WindAt2m            MetersPerSecond
	SolarRadiation      MegajoulesPerSquareMeterDay
	Pressure            Kilopascals
}

type Location struct {
	LatitudeDeg float64
	Elevation   Meters
}

func SaturationVaporPressure(t Celsius) Kilopascals {
	tc := float64(t)

	return Kilopascals(0.6108 * math.Exp(17.27*tc/(tc+237.3)))
}

func VaporPressureFromDewPoint(dewPoint Celsius) Kilopascals {
	return SaturationVaporPressure(dewPoint)
}

func VaporPressureFromHumidityExtremes(tMax, tMin Celsius, rhMax, rhMin float64) Kilopascals {
	return Kilopascals((float64(SaturationVaporPressure(tMin))*rhMax/100 + float64(SaturationVaporPressure(tMax))*rhMin/100) / 2)
}

func VaporPressureFromMeanHumidity(tMax, tMin Celsius, rhMean float64) Kilopascals {
	return Kilopascals(rhMean / 100 * meanSaturationVaporPressure(tMax, tMin))
}

func AtmosphericPressure(z Meters) Kilopascals {
	return Kilopascals(101.3 * math.Pow((293-0.0065*float64(z))/293, 5.26))
}

func PsychrometricConstant(p Kilopascals) float64 {
	return 0.665e-3 * float64(p)
}

func ExtraterrestrialRadiation(latitudeDeg float64, dayOfYear int) MegajoulesPerSquareMeterDay {
	phi := latitudeDeg * math.Pi / 180
	j := float64(dayOfYear)
	dr := 1 + 0.033*math.Cos(2*math.Pi*j/daysPerYear)
	delta := 0.409 * math.Sin(2*math.Pi*j/daysPerYear-1.39)
	ws := math.Acos(Clamp(-math.Tan(phi)*math.Tan(delta), -1, 1))
	ra := 24 * 60 / math.Pi * solarConstant * dr * (ws*math.Sin(phi)*math.Sin(delta) + math.Cos(phi)*math.Cos(delta)*math.Sin(ws))

	return MegajoulesPerSquareMeterDay(ra)
}

func ClearSkyRadiation(ra MegajoulesPerSquareMeterDay, z Meters) MegajoulesPerSquareMeterDay {
	return MegajoulesPerSquareMeterDay((0.75 + 2e-5*float64(z)) * float64(ra))
}

func NetRadiation(m DailyMeteo, ra MegajoulesPerSquareMeterDay, z Meters) MegajoulesPerSquareMeterDay {
	rs := float64(m.SolarRadiation)
	netShortwave := (1 - referenceAlbedo) * rs
	rso := float64(ClearSkyRadiation(ra, z))
	relative := 1.0

	if rso > 0 {
		relative = math.Min(rs/rso, 1)
	}

	tMaxK := math.Pow(float64(m.TMax)+kelvinOffset, 4)
	tMinK := math.Pow(float64(m.TMin)+kelvinOffset, 4)
	netLongwave := stefanBoltzmann * (tMaxK + tMinK) / 2 * (0.34 - 0.14*math.Sqrt(float64(m.ActualVaporPressure))) * (1.35*relative - 0.35)

	return MegajoulesPerSquareMeterDay(netShortwave - netLongwave)
}

func PenmanMonteith(m DailyMeteo, loc Location, dayOfYear int) Millimeters {
	tMean := (float64(m.TMax) + float64(m.TMin)) / 2
	pressure := m.Pressure

	if pressure <= 0 {
		pressure = AtmosphericPressure(loc.Elevation)
	}

	gamma := PsychrometricConstant(pressure)
	delta := slopeOfSaturationCurve(Celsius(tMean))
	ra := ExtraterrestrialRadiation(loc.LatitudeDeg, dayOfYear)
	rn := float64(NetRadiation(m, ra, loc.Elevation))
	u2 := float64(m.WindAt2m)
	vpd := meanSaturationVaporPressure(m.TMax, m.TMin) - float64(m.ActualVaporPressure)
	numerator := megajoulesToMillimeter*delta*rn + gamma*900/(tMean+273)*u2*vpd
	denominator := delta + gamma*(1+0.34*u2)

	return Millimeters(math.Max(numerator/denominator, 0))
}

func Hargreaves(tMax, tMin Celsius, ra MegajoulesPerSquareMeterDay) Millimeters {
	tMean := (float64(tMax) + float64(tMin)) / 2
	spread := math.Max(float64(tMax)-float64(tMin), 0)

	return Millimeters(math.Max(0.0023*(tMean+17.8)*math.Sqrt(spread)*megajoulesToMillimeter*float64(ra), 0))
}

func slopeOfSaturationCurve(t Celsius) float64 {
	tc := float64(t)

	return 4098 * float64(SaturationVaporPressure(t)) / ((tc + 237.3) * (tc + 237.3))
}

func meanSaturationVaporPressure(tMax, tMin Celsius) float64 {
	return (float64(SaturationVaporPressure(tMax)) + float64(SaturationVaporPressure(tMin))) / 2
}
