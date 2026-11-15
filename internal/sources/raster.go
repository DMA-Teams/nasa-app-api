package sources

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
)

var ErrUnsupportedRaster = errors.New("unsupported raster")

type Raster struct {
	Width     int
	Height    int
	OriginLon float64
	OriginLat float64
	PixelLon  float64
	PixelLat  float64
	NoData    float64
	values    []int16
}

func ReadENVI(binPath string) (*Raster, error) {
	header := strings.TrimSuffix(binPath, filepath.Ext(binPath)) + ".hdr"

	r, err := parseENVIHeader(header)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(binPath)
	if err != nil {
		return nil, fmt.Errorf("read raster %s: %w", binPath, err)
	}

	if len(data) != r.Width*r.Height*2 {
		return nil, fmt.Errorf("raster %s has %d bytes for %dx%d int16: %w", binPath, len(data), r.Width, r.Height, ErrUnsupportedRaster)
	}

	r.values = make([]int16, r.Width*r.Height)
	for i := range r.values {
		r.values[i] = int16(binary.LittleEndian.Uint16(data[2*i:]))
	}

	return r, nil
}

func parseENVIHeader(path string) (*Raster, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open header %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	r := &Raster{NoData: math.NaN()}
	fields := map[string]string{}
	sc := bufio.NewScanner(f)

	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), "="); ok {
			fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan header %s: %w", path, err)
	}

	if fields["data type"] != "2" || fields["byte order"] != "0" {
		return nil, fmt.Errorf("header %s must describe little-endian int16: %w", path, ErrUnsupportedRaster)
	}

	r.Width, _ = strconv.Atoi(fields["samples"])
	r.Height, _ = strconv.Atoi(fields["lines"])

	if v, ok := fields["data ignore value"]; ok {
		r.NoData, _ = strconv.ParseFloat(v, 64)
	}

	info := strings.Split(strings.Trim(fields["map info"], "{} "), ",")
	if len(info) < 7 || r.Width == 0 || r.Height == 0 {
		return nil, fmt.Errorf("header %s without size or map info: %w", path, ErrUnsupportedRaster)
	}

	nums := make([]float64, 6)
	for i := range nums {
		nums[i], err = strconv.ParseFloat(strings.TrimSpace(info[i+1]), 64)
		if err != nil {
			return nil, fmt.Errorf("map info value %q: %w", info[i+1], err)
		}
	}

	refX, refY := nums[0]-1, nums[1]-1
	r.PixelLon, r.PixelLat = nums[4], nums[5]
	r.OriginLon = nums[2] - refX*r.PixelLon
	r.OriginLat = nums[3] + refY*r.PixelLat

	return r, nil
}

func (r *Raster) ValuesIn(mp orb.MultiPolygon, scale float64) []float64 {
	b := mp.Bound()
	col0 := max(int((b.Min.Lon()-r.OriginLon)/r.PixelLon), 0)
	col1 := min(int((b.Max.Lon()-r.OriginLon)/r.PixelLon)+1, r.Width)
	row0 := max(int((r.OriginLat-b.Max.Lat())/r.PixelLat), 0)
	row1 := min(int((r.OriginLat-b.Min.Lat())/r.PixelLat)+1, r.Height)

	var out []float64

	for row := row0; row < row1; row++ {
		lat := r.OriginLat - (float64(row)+0.5)*r.PixelLat

		for col := col0; col < col1; col++ {
			v := float64(r.values[row*r.Width+col])
			if v == r.NoData {
				continue
			}

			lon := r.OriginLon + (float64(col)+0.5)*r.PixelLon
			if planar.MultiPolygonContains(mp, orb.Point{lon, lat}) {
				out = append(out, v/scale)
			}
		}
	}

	return out
}
