package store

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

const (
	seriesMagic         = "NAS1"
	seriesFormatVersion = 1
	seriesParameters    = 9
	seriesHeaderSize    = 4 + 2 + 2 + 1 + 1 + 4 + 1
)

var ErrCorruptSeries = errors.New("corrupt climate series")

func EncodeCellSeries(s *agro.CellSeries) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("encode cell series: %w", err)
	}

	n := s.Len()
	buf := make([]byte, seriesHeaderSize, seriesHeaderSize+seriesParameters*n*4)
	copy(buf, seriesMagic)
	binary.LittleEndian.PutUint16(buf[4:], seriesFormatVersion)
	binary.LittleEndian.PutUint16(buf[6:], uint16(s.Start.Year()))
	buf[8] = byte(s.Start.Month())
	buf[9] = byte(s.Start.Day())
	binary.LittleEndian.PutUint32(buf[10:], uint32(n))
	buf[14] = seriesParameters

	for _, column := range columns(s) {
		for i := range n {
			buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(float32(column(i))))
		}
	}

	return buf, nil
}

func DecodeCellSeries(data []byte) (*agro.CellSeries, error) {
	if len(data) < seriesHeaderSize || string(data[:4]) != seriesMagic {
		return nil, fmt.Errorf("bad header: %w", ErrCorruptSeries)
	}

	if v := binary.LittleEndian.Uint16(data[4:]); v != seriesFormatVersion {
		return nil, fmt.Errorf("format version %d: %w", v, ErrCorruptSeries)
	}

	n := int(binary.LittleEndian.Uint32(data[10:]))
	if data[14] != seriesParameters || len(data) != seriesHeaderSize+seriesParameters*n*4 {
		return nil, fmt.Errorf("length %d for %d days: %w", len(data), n, ErrCorruptSeries)
	}

	start := time.Date(int(binary.LittleEndian.Uint16(data[6:])), time.Month(data[8]), int(data[9]), 0, 0, 0, 0, time.UTC)
	s := newCellSeries(start, n)
	values := make([]float64, n)
	offset := seriesHeaderSize

	for _, assign := range assigners(s) {
		for i := range n {
			values[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset:])))
			offset += 4
		}

		assign(values)
	}

	return s, nil
}

func newCellSeries(start time.Time, n int) *agro.CellSeries {
	return &agro.CellSeries{
		Start:          start,
		Precipitation:  make([]agro.Millimeters, n),
		TMean:          make([]agro.Celsius, n),
		TMax:           make([]agro.Celsius, n),
		TMin:           make([]agro.Celsius, n),
		RelativeHumid:  make([]float64, n),
		DewPoint:       make([]agro.Celsius, n),
		Wind:           make([]agro.MetersPerSecond, n),
		SolarRadiation: make([]agro.MegajoulesPerSquareMeterDay, n),
		Pressure:       make([]agro.Kilopascals, n),
	}
}

func columns(s *agro.CellSeries) [seriesParameters]func(int) float64 {
	return [seriesParameters]func(int) float64{
		func(i int) float64 { return float64(s.Precipitation[i]) },
		func(i int) float64 { return float64(s.TMean[i]) },
		func(i int) float64 { return float64(s.TMax[i]) },
		func(i int) float64 { return float64(s.TMin[i]) },
		func(i int) float64 { return s.RelativeHumid[i] },
		func(i int) float64 { return float64(s.DewPoint[i]) },
		func(i int) float64 { return float64(s.Wind[i]) },
		func(i int) float64 { return float64(s.SolarRadiation[i]) },
		func(i int) float64 { return float64(s.Pressure[i]) },
	}
}

func assigners(s *agro.CellSeries) [seriesParameters]func([]float64) {
	return [seriesParameters]func([]float64){
		func(v []float64) { convertInto(s.Precipitation, v) },
		func(v []float64) { convertInto(s.TMean, v) },
		func(v []float64) { convertInto(s.TMax, v) },
		func(v []float64) { convertInto(s.TMin, v) },
		func(v []float64) { copy(s.RelativeHumid, v) },
		func(v []float64) { convertInto(s.DewPoint, v) },
		func(v []float64) { convertInto(s.Wind, v) },
		func(v []float64) { convertInto(s.SolarRadiation, v) },
		func(v []float64) { convertInto(s.Pressure, v) },
	}
}

func convertInto[T ~float64](dst []T, src []float64) {
	for i, v := range src {
		dst[i] = T(v)
	}
}
