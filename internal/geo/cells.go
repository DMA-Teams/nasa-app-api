package geo

import (
	"fmt"
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/clip"
	"github.com/paulmach/orb/planar"
)

const (
	PowerCellLatStep = 0.5
	PowerCellLonStep = 0.625
	minimumCellShare = 0.02
)

type Cell struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (c Cell) ID() string {
	return fmt.Sprintf("%.2f_%.3f", c.Lat, c.Lon)
}

func (c Cell) Bound() orb.Bound {
	return orb.Bound{
		Min: orb.Point{c.Lon - PowerCellLonStep/2, c.Lat - PowerCellLatStep/2},
		Max: orb.Point{c.Lon + PowerCellLonStep/2, c.Lat + PowerCellLatStep/2},
	}
}

func PowerCellOf(lat, lon float64) Cell {
	return Cell{
		Lat: math.Round(lat/PowerCellLatStep) * PowerCellLatStep,
		Lon: math.Round(lon/PowerCellLonStep) * PowerCellLonStep,
	}
}

func PowerCellsCovering(b orb.Bound) []Cell {
	minCell := PowerCellOf(b.Min.Lat(), b.Min.Lon())
	maxCell := PowerCellOf(b.Max.Lat(), b.Max.Lon())

	var out []Cell

	for lat := minCell.Lat; lat <= maxCell.Lat+1e-9; lat += PowerCellLatStep {
		for lon := minCell.Lon; lon <= maxCell.Lon+1e-9; lon += PowerCellLonStep {
			out = append(out, PowerCellOf(lat, lon))
		}
	}

	return out
}

type CellShare struct {
	Cell  Cell
	Share float64
}

func CellShares(mp orb.MultiPolygon) []CellShare {
	total := planar.Area(mp)
	if total <= 0 {
		c := mp.Bound().Center()

		return []CellShare{{Cell: PowerCellOf(c.Lat(), c.Lon()), Share: 1}}
	}

	var shares []CellShare

	sum := 0.0

	for _, cell := range PowerCellsCovering(mp.Bound()) {
		part := clip.MultiPolygon(cell.Bound(), mp.Clone())
		if len(part) == 0 {
			continue
		}

		share := planar.Area(part) / total
		if share >= minimumCellShare {
			shares = append(shares, CellShare{Cell: cell, Share: share})
			sum += share
		}
	}

	if len(shares) == 0 {
		c := mp.Bound().Center()

		return []CellShare{{Cell: PowerCellOf(c.Lat(), c.Lon()), Share: 1}}
	}

	for i := range shares {
		shares[i].Share /= sum
	}

	return shares
}

func SamplePoints(mp orb.MultiPolygon, spacingDeg float64, minimum int) []orb.Point {
	for spacing := spacingDeg; spacing > spacingDeg/27; spacing /= 3 {
		points := gridInside(mp, spacing)
		if len(points) >= minimum {
			return points
		}
	}

	c, _ := planar.CentroidArea(mp)
	if planar.MultiPolygonContains(mp, c) {
		return []orb.Point{c}
	}

	return []orb.Point{mp[0][0][0]}
}

func gridInside(mp orb.MultiPolygon, spacing float64) []orb.Point {
	b := mp.Bound()

	var out []orb.Point

	for lat := b.Min.Lat() + spacing/2; lat < b.Max.Lat(); lat += spacing {
		for lon := b.Min.Lon() + spacing/2; lon < b.Max.Lon(); lon += spacing {
			p := orb.Point{lon, lat}
			if planar.MultiPolygonContains(mp, p) {
				out = append(out, p)
			}
		}
	}

	return out
}

func InteriorPoint(mp orb.MultiPolygon) orb.Point {
	c, _ := planar.CentroidArea(mp)
	if planar.MultiPolygonContains(mp, c) {
		return c
	}

	return SamplePoints(mp, mp.Bound().Max.Lat()-mp.Bound().Min.Lat(), 1)[0]
}
