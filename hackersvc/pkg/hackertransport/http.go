package hackertransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"

	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	httptransport "github.com/go-kit/kit/transport/http"

	"github.com/ctfsys/backend/hackersvc/pkg/hackerendpoint"
	"github.com/ctfsys/backend/hackersvc/pkg/hackerservice"
)

// NewHTTPHandler returns an HTTP handler that makes a set of endpoints
// available on predifined paths.
func NewHTTPHandler(endpoints hackerendpoint.Set, tracer stdopentracing.Tracer, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorLogger(logger),
	}
	m := http.NewServeMux()

	m.Handle("/ping", httptransport.NewServer(
		endpoints.PingEndpoint,
		decodeHTTPPingRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "Ping", logger)))...,
	))

	return m
}

// NewHTTPClient returns a HackerService backed by an HTTP server living at the
// remote instance. We expect instance to come from a service discovery system,
// so likely of the form "host:port". We bake-in certain middlewares,
// implementing the client library pattern.
func NewHTTPClient(instance string, tracer stdopentracing.Tracer, logger log.Logger) (hackerservice.Service, error) {
	// Quickly sanitize the instance string.
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))

	var pingEndpoint endpoint.Endpoint
	{
		pingEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, "/ping"),
			encodeHTTPGenericRequest,
			decodeHTTPPingResponse,
			httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
		).Endpoint()
		pingEndpoint = opentracing.TraceClient(tracer, "Ping")(pingEndpoint)
		pingEndpoint = limiter(pingEndpoint)
		pingEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Ping",
			Timeout: 30 * time.Second,
		}))(pingEndpoint)
	}

	return hackerendpoint.Set{
		PingEndpoint: pingEndpoint,
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

func errorEncoder(_ context.Context, err error, rw http.ResponseWriter) {
	rw.WriteHeader(err2code(err))
	json.NewEncoder(rw).Encode(errorWrapper{Error: err.Error()})
}

func err2code(err error) int {
	switch err {
	case hackerservice.ErrRandomFailure:
		// ErrRandomFailure is a dummy error that is randomly returned from Ping().
		// We return a "dummy" error when we get that.
		return http.StatusTeapot
	}
	return http.StatusInternalServerError
}

func errorDecoder(req *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(req.Body).Decode(&w); err != nil {
		return err
	}

	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string `json:"error"`
}

// decodeHTTPPingRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded ping request from the HTTP request body. Primarily useful in a
// server.
func decodeHTTPPingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req hackerendpoint.PingRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// decodeHTTPPingResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded ping response from the HTTP response body. If the response has a
// non-200 status code, we will interpret that as an error and attempt to decode
// the specific error message from the response body. Primarily useful in a
// client.
func decodeHTTPPingResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}

	var resp hackerendpoint.PingResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// encodeHTTPGenericRequest is a transport/http.EncodeRequestFunc tht
// JSON-encodes any request to the request body. Primarily useful in a client.
func encodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}

	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// encodeHTTPGenericResponse is a transport/http.EncodeReponseFunc that encodes
// the response as JSON to the response writer. Primarily userful in a server.
func encodeHTTPGenericResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	if f, ok := response.(hackerendpoint.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), rw)
		return nil
	}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(rw).Encode(response)
}
