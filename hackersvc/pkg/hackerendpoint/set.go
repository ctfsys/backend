package hackerendpoint

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"

	"github.com/ctfsys/backend/hackersvc/pkg/hackerservice"
)

// Set collects all of the endpoints that compose a hacker service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
	PingEndpoint endpoint.Endpoint
}

// New returns a Set that wraps the provided server, and wires in all of hte
// expected endpoint middleware via the various parameters.
func New(
	svc hackerservice.Service,
	logger log.Logger,
	duration metrics.Histogram,
	trace stdopentracing.Tracer,
) Set {
	var pingEndpoint endpoint.Endpoint
	{
		pingEndpoint = MakePingEndpoint(svc)
		pingEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))(pingEndpoint)
		pingEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(pingEndpoint)
		pingEndpoint = opentracing.TraceServer(trace, "Ping")(pingEndpoint)
		pingEndpoint = LoggingMiddleware(log.With(logger, "method", "Ping"))(pingEndpoint)
		pingEndpoint = InstrumentingMiddleware(duration.With("method", "Ping"))(pingEndpoint)
	}

	return Set{
		PingEndpoint: pingEndpoint,
	}
}

// Ping implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) Ping(ctx context.Context) (string, error) {
	resp, err := s.PingEndpoint(ctx, PingRequest{})
	if err != nil {
		return "", err
	}
	response := resp.(PingResponse)
	return response.P, response.Err
}

// MakePingEndpoint constructs a Ping endpoint wrapping the service.
func MakePingEndpoint(s hackerservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		// Ping doesn't take arguments, but here you would use req.Field
		// as arguments to ping.
		_ = request.(PingRequest)
		p, err := s.Ping(ctx)
		return PingResponse{P: p, Err: err}, nil
	}
}

// Failer is an interface that should be implemented by response types.
// Response encoders can check if responses are Failer, and if so if they've
// failed, and if so encode them using a separate write path based on the error.
type Failer interface {
	Failed() error
}

// PingRequest collects the request parameters for the Ping method.
type PingRequest struct{}

// PingResponse collects the response values for the Ping method.
type PingResponse struct {
	P   string `json:"p"`
	Err error  `json:"-"` // should be intercepted by Failed/errorEncoder
}

// Failed implements Failer.
func (r PingResponse) Failed() error { return r.Err }
