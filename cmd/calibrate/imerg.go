package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/sources"
)

const (
	cmrGranulesURL     = "https://cmr.earthdata.nasa.gov/search/granules.json"
	imergCollection    = "C2723754864-GES_DISC"
	imergOpendapHost   = "opendap.earthdata.nasa.gov"
	imergConstraint    = "/precipitation[0][909:921][1037:1052];/lat[1037:1052];/lon[909:921]"
	cmrPageSize        = 400
	progressEveryFiles = 250
)

type imergDownloader struct {
	cmr      *sources.Client
	data     *sources.Client
	cmrURL   string
	dir      string
	parallel int
	logger   *slog.Logger
}

type granule struct {
	day     time.Time
	opendap string
}

type cmrResponse struct {
	Feed struct {
		Entry []struct {
			TimeStart string `json:"time_start"`
			Links     []struct {
				Href string `json:"href"`
			} `json:"links"`
		} `json:"entry"`
	} `json:"feed"`
}

func (d *imergDownloader) Run(ctx context.Context, start, end time.Time) error {
	var missing []granule

	for year := start.Year(); year <= end.Year(); year++ {
		from := maxTime(start, time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC))
		to := minTime(end, time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC))

		gs, err := d.granules(ctx, from, to)
		if err != nil {
			return err
		}

		for _, g := range gs {
			if _, err := os.Stat(d.path(g.day)); errors.Is(err, fs.ErrNotExist) {
				missing = append(missing, g)
			}
		}
	}

	d.logger.Info("imerg granules to download", "missing", len(missing))

	return d.download(ctx, missing)
}

func (d *imergDownloader) granules(ctx context.Context, from, to time.Time) ([]granule, error) {
	q := url.Values{}
	q.Set("collection_concept_id", imergCollection)
	q.Set("temporal", from.Format(time.DateOnly)+"T00:00:00Z,"+to.Format(time.DateOnly)+"T23:59:59Z")
	q.Set("page_size", strconv.Itoa(cmrPageSize))
	q.Set("sort_key", "start_date")

	var resp cmrResponse
	if err := d.cmr.GetJSON(ctx, d.cmrURL+"?"+q.Encode(), &resp); err != nil {
		return nil, fmt.Errorf("list imerg granules %d: %w", from.Year(), err)
	}

	out := make([]granule, 0, len(resp.Feed.Entry))

	for _, e := range resp.Feed.Entry {
		day, err := time.Parse(time.RFC3339, e.TimeStart)
		if err != nil {
			return nil, fmt.Errorf("parse granule start %q: %w", e.TimeStart, err)
		}

		for _, l := range e.Links {
			if u, err := url.Parse(l.Href); err == nil && u.Host == imergOpendapHost {
				out = append(out, granule{day: day.UTC().Truncate(24 * time.Hour), opendap: l.Href})

				break
			}
		}
	}

	return out, nil
}

func (d *imergDownloader) download(ctx context.Context, gs []granule) error {
	jobs := make(chan granule)
	errs := make(chan error, d.parallel)

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		done int
	)

	for range d.parallel {
		wg.Go(func() {
			for g := range jobs {
				if err := d.fetch(ctx, g); err != nil {
					errs <- err

					return
				}

				mu.Lock()
				done++
				if done%progressEveryFiles == 0 {
					d.logger.Info("imerg progress", "done", done, "total", len(gs))
				}
				mu.Unlock()
			}
		})
	}

	var firstErr error

feed:
	for _, g := range gs {
		select {
		case jobs <- g:
		case firstErr = <-errs:
			break feed
		case <-ctx.Done():
			firstErr = ctx.Err()

			break feed
		}
	}

	close(jobs)
	wg.Wait()
	close(errs)

	if firstErr == nil {
		firstErr = <-errs
	}

	return firstErr
}

func (d *imergDownloader) fetch(ctx context.Context, g granule) error {
	body, err := d.data.Get(ctx, g.opendap+".dap.csv?dap4.ce="+url.QueryEscape(imergConstraint))
	if err != nil {
		return fmt.Errorf("imerg %s: %w", g.day.Format(time.DateOnly), err)
	}

	if !strings.Contains(string(body), "/precipitation[0][0]") {
		return fmt.Errorf("imerg %s: unexpected response %.80q", g.day.Format(time.DateOnly), body)
	}

	return writeAtomic(d.path(g.day), body)
}

func (d *imergDownloader) path(day time.Time) string {
	return filepath.Join(d.dir, "imerg", day.Format("2006"), day.Format("20060102")+".csv")
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}

	return nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}

	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}

	return b
}
