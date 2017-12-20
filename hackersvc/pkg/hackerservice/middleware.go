package hackerservice

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

// Middleware describes a service (as opposed to endpoint) middleware.
type Middleware func(Service) Service

// LoggingMiddleware takes a logger as a dependency
// and returns a ServiceMiddleware.
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) Ping(ctx context.Context) (p string, err error) {
	defer func() {
		mw.logger.Log("method", "Ping", "p", p, "err", err)
	}()
	return mw.next.Ping(ctx)
}

// InstrumentingMiddleware returns a service middleware that instruments
// the number of pings asked for over the lifetime of the service.
func InstrumentingMiddleware(pings metrics.Counter) Middleware {
	return func(next Service) Service {
		return instrumentingMiddleware{
			pings: pings,
			next:  next,
		}
	}
}

type instrumentingMiddleware struct {
	pings metrics.Counter
	next  Service
}

func (mw instrumentingMiddleware) Ping(ctx context.Context) (string, error) {
	p, err := mw.next.Ping(ctx)
	mw.pings.Add(float64(1))
	return p, err
}
