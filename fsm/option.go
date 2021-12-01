package fsm

import (
	"time"
)

type Options struct {
	// how many connections we will pool
	maxPool int
	// the amount of time we wait for the command to be started.
	timeout time.Duration
	// how many snapshots are retained.
	retain int
}

func (o *Options) setDefault() {
	o.maxPool = 3
	o.retain = 2
	o.timeout = 10 * time.Second
}

type Option func(*Options)

func newOptions(opt ...Option) *Options {
	o := &Options{}
	o.setDefault()
	for _, f := range opt {
		f(o)
	}
	return o
}

func noop(_ *Options) {}

// WithMaxPool sets a parameter controls how many connections we will pool.
// Default is 3.
func WithMaxPool(p int) Option {
	if p < 1 {
		return noop
	}
	return func(o *Options) {
		o.maxPool = p
	}
}

// WithRetain sets a parameter controls how many snapshots are retained.
// If it is less than 1, sets 1 forcefully. retain must be at least 1.
// Default is 2.
func WithRetain(r int) Option {
	if r < 1 {
		return noop
	}
	return func(o *Options) {
		o.retain = r
	}
}

// WithTimeoutSecond sets a parameter limits the amount of time
// we wait for the command to be started.
// Default is 10 seconds.
func WithTimeoutSecond(t int) Option {
	if t < 1 {
		return noop
	}
	return func(o *Options) {
		o.timeout = time.Duration(t) * time.Second
	}
}
