package store

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

type VersionReader interface {
	CurrentVersions(ctx context.Context) (Versions, error)
}

type Watcher struct {
	reader   VersionReader
	interval time.Duration
	timeout  time.Duration
	current  atomic.Pointer[Versions]
	onChange func(ctx context.Context, v Versions) error
	logger   *slog.Logger
}

func NewWatcher(reader VersionReader, interval, timeout time.Duration, onChange func(context.Context, Versions) error, logger *slog.Logger) *Watcher {
	return &Watcher{reader: reader, interval: interval, timeout: timeout, onChange: onChange, logger: logger}
}

func (w *Watcher) Current() (Versions, bool) {
	v := w.current.Load()
	if v == nil {
		return Versions{}, false
	}

	return *v, true
}

func (w *Watcher) Refresh(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	v, err := w.reader.CurrentVersions(ctx)
	if err != nil {
		return err
	}

	if old := w.current.Load(); old != nil && *old == v {
		return nil
	}

	if w.onChange != nil {
		if err := w.onChange(ctx, v); err != nil {
			return err
		}
	}

	w.current.Store(&v)

	return nil
}

func (w *Watcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.Refresh(ctx); err != nil {
				w.logger.WarnContext(ctx, "version refresh failed", slog.String("error", err.Error()))
			}
		}
	}
}
