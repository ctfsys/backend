package hackertransport

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	"github.com/sony/gobreaker"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/ratelimit"

	"github.com/ctfsys/backend/hackersvc/pkg/hackerendpoint"
	"github.com/ctfsys/backend/hackersvc/pkg/hackerservice"
	hackerthrift "github.com/ctfsys/backend/hackersvc/thrift/gen-go/hackersvc"
)

type thriftServer struct {
	ctx       context.Context
	endpoints hackerendpoint.Set
}

// NewThriftServer makes a set of endpoints available as a Thrift service.
func NewThriftServer(endpoints hackerendpoint.Set) hackerthrift.HackerService {
	return &thriftServer{
		endpoints: endpoints,
	}
}

func (s *thriftServer) Ping(ctx context.Context) (*hackerthrift.PingReply, error) {
	request := hackerendpoint.PingRequest{}
	response, err := s.endpoints.PingEndpoint(ctx, request)
	if err != nil {
		return nil, err
	}

	resp := response.(hackerendpoint.PingResponse)
	return &hackerthrift.PingReply{Value: resp.P, Err: err2str(resp.Err)}, nil
}

// NewThriftClient returns a HackerService backed by a Thrift server described by
// the provided client. The caller is responsible for constructing the client,
// and eventually closing the underlying transport. We bake-in certain middlewares,
// implementing the client library pattern.
func NewThriftClient(client *hackerthrift.HackerServiceClient) hackerservice.Service {
	// We construct a single ratelimiter middleware, to limit the total
	// outgoing QPS from this client to all methods on the remote instance. We
	// also construct per-endpoint circuitbreaker middlewares, although they
	// could easily be combined into a single breaker for the entire remote
	// instance, too.
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))

	// Each individual endpoint is an http/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
	var pingEndpoint endpoint.Endpoint
	{
		pingEndpoint = MakeThriftPingEndpoint(client)
		pingEndpoint = limiter(pingEndpoint)
		pingEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Ping",
			Timeout: 10 * time.Second,
		}))(pingEndpoint)
	}

	// Returning the endpoint.Set as a service.Service relies on the
	// endpoint.Set implementing the Service methods. That's just a simple bit
	// of glue code.
	return hackerendpoint.Set{
		PingEndpoint: pingEndpoint,
	}
}

// MakeThriftPingEndpoint returns an endpoint that invokes the passed Thrift
// client.  Useful only in clients, and only until a proper
// go-kit/kit/transport/thrift.Client exists.
func MakeThriftPingEndpoint(client *hackerthrift.HackerServiceClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Ping doesn't take any arguments, but here you would pass request
		// parameters to Ping
		_ = request.(hackerendpoint.PingRequest)
		reply, err := client.Ping(ctx)
		if err == hackerservice.ErrRandomFailure {
			return nil, err // special case; see comment on ErrRandomFailure
		}
		return hackerendpoint.PingResponse{P: reply.Value, Err: err}, nil
	}
}
