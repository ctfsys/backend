package hackertransport

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	oldcontext "golang.org/x/net/context"
	"golang.org/x/time/rate"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"

	"github.com/ctfsys/backend/hackersvc/pb"
	"github.com/ctfsys/backend/hackersvc/pkg/hackerendpoint"
	"github.com/ctfsys/backend/hackersvc/pkg/hackerservice"
)

type grpcServer struct {
	ping grpctransport.Handler
}

// NewGRPCServer makes a set of endpoints available as a gRPC HackerServer.
func NewGRPCServer(endpoints hackerendpoint.Set, tracer stdopentracing.Tracer, logger log.Logger) pb.HackerServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		ping: grpctransport.NewServer(
			endpoints.PingEndpoint,
			decodeGRPCPingRequest,
			encodeGRPCPingResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(tracer, "Ping", logger)))...,
		),
	}
}

func (s *grpcServer) Ping(ctx oldcontext.Context, req *pb.PingRequest) (*pb.PingReply, error) {
	_, rep, err := s.ping.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.PingReply), nil
}

// NewGRPCClient returns a HackerService backed by a gRPC server at the other end
// of the conn. The caller is responsible for constructing the conn, and
// eventually closing the underlying transport. We bake-in certain middlewares,
// implementing the client library pattern.
func NewGRPCClient(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) hackerservice.Service {
	// We construct a single ratelimiter middleware, to limit the total
	// outgoing QPS from this client to all methods on the remote instance. We
	// also construct per-endpoint circuitbreaker middlewares, although they
	// could easily be combined into a single breaker for the entire remote
	// instance, too.
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(
		rate.Every(time.Second), 100))

	// Each individual endpoint is an http/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middleware. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consisten set of blient behaviour.
	var pingEndpoint endpoint.Endpoint
	{
		pingEndpoint = grpctransport.NewClient(
			conn,
			"pb.Hacker",
			"Ping",
			encodeGRPCPingRequest,
			decodeGRPCPingResponse,
			pb.PingReply{},
			grpctransport.ClientBefore(opentracing.ContextToGRPC(tracer, logger)),
		).Endpoint()
		pingEndpoint = opentracing.TraceClient(tracer, "Ping")(pingEndpoint)
		pingEndpoint = limiter(pingEndpoint)
		pingEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Ping",
			Timeout: 30 * time.Second,
		}))(pingEndpoint)
	}

	// Returning the endpoint.Set as a service.Service relies on the
	// endpoint.Set implementing the Service methods.
	return hackerendpoint.Set{
		PingEndpoint: pingEndpoint,
	}
}

// decodeGRPCPingRequest is a transport/grpc.DecodeRequestFunc that converts a
// gRPC ping request to a user-domain ping request. Primarily useful in a server.
func decodeGRPCPingRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	// Ping doesn't take any arguments, but here you would pass parameters from
	// the request into PingRequest.
	_ = grpcReq.(*pb.PingRequest)
	return hackerendpoint.PingRequest{}, nil
}

// decodeGRPCPingResponse is a transport/grpc.DecodeResponseFunc that converts a
// gRPC ping reply to a user-domain ping response. Primarily useful in a client.
func decodeGRPCPingResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.PingReply)
	return hackerendpoint.PingResponse{
		P:   string(reply.P),
		Err: str2err(reply.Err),
	}, nil
}

// encodeGRPCPingResponse is a transport/grpc.EncodeResponseFunc that converts a
// user-domain ping response to a gRPC ping reply. Primarily useful in a server.
func encodeGRPCPingResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(hackerendpoint.PingResponse)
	return &pb.PingReply{P: string(resp.P), Err: err2str(resp.Err)}, nil
}

// encodeGRPCPingRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain ping request to a gRPC ping request. Primarily userful in a client.
func encodeGRPCPingRequest(_ context.Context, request interface{}) (interface{}, error) {
	_ = request.(hackerendpoint.PingRequest)
	return &pb.PingRequest{}, nil
}

// These annoying helper functions are required to translate Go error types to
// and from strings, which is the type we use in our IDLs to represent errors.
// There is special casing to treat empty strings as nil errors.

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
