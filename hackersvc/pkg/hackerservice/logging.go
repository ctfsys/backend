package hackerservice

import (
	"context"

	"github.com/go-kit/kit/log"
)

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
