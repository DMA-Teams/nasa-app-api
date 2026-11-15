package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dma-teams/nasa-app-api/internal/sources"
)

type fakeOpendap struct {
	calls atomic.Int32
}

func (f *fakeOpendap) RoundTrip(r *http.Request) (*http.Response, error) {
	f.calls.Add(1)

	body := "Dataset: x\n/lon, -88.95\n/lat, 13.85\n/precipitation[0][0], 1.5\n"
	if r.Header.Get("Authorization") != "Bearer token" || !strings.Contains(r.URL.RawQuery, "dap4.ce=") {
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	}

	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func cmrServer(t *testing.T, days ...string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("collection_concept_id") != imergCollection {
			http.Error(w, "bad collection", http.StatusBadRequest)

			return
		}

		year := r.URL.Query().Get("temporal")[:4]
		entries := make([]string, 0, len(days))

		for _, d := range days {
			if !strings.HasPrefix(d, year) {
				continue
			}

			entries = append(entries, fmt.Sprintf(`{"time_start":"%sT00:00:00.000Z","links":[{"href":"https://data.example/x.nc4"},{"href":"https://%s/collections/C/granules/%s.nc4"}]}`, d, imergOpendapHost, d))
		}

		_, _ = fmt.Fprintf(w, `{"feed":{"entry":[%s]}}`, strings.Join(entries, ","))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestImergDownloader_DownloadsMissingDaysOnly(t *testing.T) {
	dir := t.TempDir()
	cmr := cmrServer(t, "2015-07-01", "2015-07-02", "2016-01-01")
	data := &fakeOpendap{}

	existing := filepath.Join(dir, "imerg", "2015", "20150701.csv")
	if err := writeAtomic(existing, []byte("kept")); err != nil {
		t.Fatal(err)
	}

	d := &imergDownloader{
		cmr:      sources.NewClient(time.Second),
		data:     sources.NewClient(time.Second, sources.WithTransport(data), sources.WithBearer("token")),
		cmrURL:   cmr.URL,
		dir:      dir,
		parallel: 2,
		logger:   slog.New(slog.DiscardHandler),
	}

	start := time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC)

	if err := d.Run(t.Context(), start, end); err != nil {
		t.Fatal(err)
	}

	if got := data.calls.Load(); got != 2 {
		t.Errorf("data calls = %d, want 2", got)
	}

	if b, _ := os.ReadFile(existing); string(b) != "kept" {
		t.Errorf("existing file overwritten: %q", b)
	}

	for _, f := range []string{"2015/20150702.csv", "2016/20160101.csv"} {
		b, err := os.ReadFile(filepath.Join(dir, "imerg", filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(b), "/precipitation[0][0]") {
			t.Errorf("%s: unexpected content %q", f, b)
		}
	}
}

func TestImergDownloader_RejectsUnexpectedBody(t *testing.T) {
	dir := t.TempDir()
	cmr := cmrServer(t, "2015-07-01")

	d := &imergDownloader{
		cmr:      sources.NewClient(time.Second),
		data:     sources.NewClient(time.Second, sources.WithTransport(&fakeOpendap{})),
		cmrURL:   cmr.URL,
		dir:      dir,
		parallel: 1,
		logger:   slog.New(slog.DiscardHandler),
	}

	day := time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)
	if err := d.Run(t.Context(), day, day); err == nil {
		t.Fatal("want error for unauthorized response")
	}

	if _, err := os.Stat(d.path(day)); err == nil {
		t.Error("file written for failed download")
	}
}
