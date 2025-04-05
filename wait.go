package wait

import (
	"context"
	"log"
	"time"

	"github.com/memsql/errors"
)

type O struct {
	TimeLimit     time.Duration
	StartInterval time.Duration
	MaxInterval   time.Duration
	Logger        Logger
	Backoff       float64
	Reports       int
	Reporter      Reporter
	Description   string
	Ctx           context.Context
	ExitOnError   bool
}

type Logger func(fmt string, args ...any)

type Reporter func(opts O, startTime time.Time)

type Option func(*O)

// WithLimit sets the maximum total time to try
func WithLimit(d time.Duration) Option { return func(o *O) { o.TimeLimit = d } }

func WithMinInterval(d time.Duration) Option { return func(o *O) { o.StartInterval = d } }
func WithMaxInterval(d time.Duration) Option { return func(o *O) { o.MaxInterval = d } }
func WithLogger(f Logger) Option             { return func(o *O) { o.Logger = f } }
func WithReporter(f Reporter) Option         { return func(o *O) { o.Reporter = f } }
func WithDescription(s string) Option        { return func(o *O) { o.Description = s } }
func WithContext(ctx context.Context) Option { return func(o *O) { o.Ctx = ctx } }
func ExitOnError(exit bool) Option           { return func(o *O) { o.ExitOnError = exit } }

// WithBackoff sets how much the interval should change after the function
// is called. Reasonable values are in the range 1.01 to 1.04. The default is
// 1.02. Do not set this to a value that is < 1.0.
func WithBackoff(f float64) Option { return func(o *O) { o.Backoff = f } }

// WithReports specifies that there be approximately N progress reports before timeout.
// Values must be 0 and above.
func WithReports(n int) Option { return func(o *O) { o.Reports = n } }

// WithInterval sets both the minimum and maximum intervals
func WithInterval(d time.Duration) Option {
	return func(o *O) {
		o.StartInterval = d
		o.MaxInterval = d
	}
}

const ErrTimeout errors.String = "timeout"

func defaultReporter(opts O, startTime time.Time) {
	opts.Logger("%s-%s wait for %s, in progress", startTime.UTC().Format("15:04:05"), time.Now().UTC().Format("15:04:05"), opts.Description)
}

// For calls a function repeatedly.
// It will stop calling and return ErrTimeout if too much time has passed.
// Otherwise it stops when the function returns true.
// If the function returns (true, error), then For() returns that error.
// An error return from function parameter does not cause the loop to exit unless ExitOnError(true) is is set.
func For(f func() (bool, error), options ...Option) error {
	initialOpts := &O{
		TimeLimit:     time.Minute * 30,
		StartInterval: time.Second,
		MaxInterval:   time.Minute,
		Backoff:       1.02,
		Logger:        log.Printf,
		Reports:       30,
		Reporter:      defaultReporter,
		Description:   "condition",
	}

	for _, opt := range options {
		opt(initialOpts)
	}

	opts := *initialOpts

	startTime := time.Now()
	limit := startTime.Add(opts.TimeLimit)
	prior := startTime
	interval := opts.StartInterval
	var reportsGiven int

	for {
		ok, err := f()
		if ok {
			// propagate error, if any
			return err
		}
		if err != nil && opts.ExitOnError {
			return err
		}

		now := time.Now()

		if now.After(limit) {
			if err != nil {
				return ErrTimeout.Errorf("%s to %s wait for %s gave up after %s: %w",
					startTime.Format("15:04:05"), now.Format("15:04:05"), opts.Description, now.Sub(startTime), err)
			} else {
				return ErrTimeout.Errorf("%s to %s wait for %s gave up after %s: not ok",
					startTime.Format("15:04:05"), now.Format("15:04:05"), opts.Description, now.Sub(startTime))
			}
		}

		if opts.Reports > 0 && float64(reportsGiven+1)/float64(opts.Reports+1) < float64(now.Sub(startTime))/float64(limit.Sub(startTime)) {
			opts.Reporter(opts, startTime)
			reportsGiven++
			now = time.Now()
		}

		next := prior.Add(interval)
		prior = now
		interval = time.Duration(opts.Backoff * float64(interval))
		if interval > opts.MaxInterval {
			interval = opts.MaxInterval
		}
		if next.After(limit) {
			next = limit
		}
		thisSleep := next.Sub(now)
		if thisSleep < 0 {
			continue
		}

		if opts.Ctx != nil {
			select {
			case <-opts.Ctx.Done():
				return opts.Ctx.Err()
			case <-time.After(thisSleep):
				//
			}
		} else {
			time.Sleep(thisSleep)
		}
	}
}
