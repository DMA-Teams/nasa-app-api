package plan

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/dma-teams/nasa-app-api/internal/agro"
)

func (s *Service) DepartmentPlan(ctx context.Context, code string, opts Options) (*DepartmentPlan, error) {
	versions, err := s.versions()
	if err != nil {
		return nil, err
	}

	dep, err := s.deps.Gazetteer.Department(code)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", code, ErrDepartmentNotFound)
	}

	codes, err := s.deps.Gazetteer.MunicipalityCodesOf(code)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", code, ErrDepartmentNotFound)
	}

	plans := make([]*MunicipalPlan, len(codes))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(s.cfg.MaxParallel)

	for i, c := range codes {
		g.Go(func() error {
			p, err := s.MunicipalPlan(gctx, c, opts)
			if err != nil {
				return err
			}

			plans[i] = p

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("department %s: %w", code, err)
	}

	return &DepartmentPlan{
		Department:     Municipality{Code: dep.Code, Name: dep.Name, DepartmentCode: dep.Code, DepartmentName: dep.Name},
		GeneratedAt:    s.cfg.Now().UTC().Format(time.RFC3339),
		Municipalities: len(plans),
		Unavailable:    s.deps.Gazetteer.UnavailableCountOf(code),
		Groups:         groupByRecommendation(plans),
		Versions:       versions,
		Warnings:       mergedWarnings(plans),
		Disclaimer:     Disclaimer,
	}, nil
}

func groupByRecommendation(plans []*MunicipalPlan) []Group {
	index := map[string]int{}

	var groups []Group

	for _, p := range plans {
		best, ok := bestRotation(p)
		if !ok {
			continue
		}

		maxLoss := 0.0
		crops := make([]string, len(best.Plantings))

		for i, pl := range best.Plantings {
			crops[i] = pl.CropID
			maxLoss = math.Max(maxLoss, pl.LossProbability)
		}

		i, seen := index[best.Signature]
		if !seen {
			i = len(groups)
			index[best.Signature] = i
			groups = append(groups, Group{
				Signature: best.Signature, RotationName: best.Name, Crops: crops,
				TotalScore: Range{Min: math.Inf(1), Max: math.Inf(-1)}, MaxLossProbability: Range{Min: math.Inf(1), Max: math.Inf(-1)},
			})
		}

		g := &groups[i]
		g.Municipalities = append(g.Municipalities, p.Municipality)
		g.TotalScore = widen(g.TotalScore, best.Scores.Total)
		g.MaxLossProbability = widen(g.MaxLossProbability, maxLoss)
	}

	slices.SortStableFunc(groups, func(a, b Group) int { return cmp.Compare(len(b.Municipalities), len(a.Municipalities)) })

	return groups
}

func widen(r Range, v float64) Range {
	return Range{Min: math.Min(r.Min, v), Max: math.Max(r.Max, v)}
}

func bestRotation(p *MunicipalPlan) (agro.RotationResult, bool) {
	for _, r := range p.Evaluation.Rotations {
		if r.ID == p.Evaluation.Best {
			return r, true
		}
	}

	return agro.RotationResult{}, false
}

func mergedWarnings(plans []*MunicipalPlan) []string {
	seen := map[string]bool{}
	out := []string{}

	for _, p := range plans {
		for _, w := range p.Warnings {
			if !seen[w] {
				seen[w] = true
				out = append(out, w)
			}
		}
	}

	slices.Sort(out)

	return out
}
