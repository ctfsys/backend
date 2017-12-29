package hackerservice

import (
	"context"

	"github.com/go-kit/kit/metrics"
)

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
