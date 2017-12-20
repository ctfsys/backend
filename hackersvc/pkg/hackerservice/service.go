package hackerservice

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

// Service describes a service that represents hackers
type Service interface {
	// Ping should return "pong" each time
	Ping(ctx context.Context) (string, error)
}

// New returns a basic Service with all of the expected middleware wired in.
func New(logger log.Logger, pings metrics.Counter) Service {
	var svc Service
	{
		svc = NewBasicService()
		svc = LoggingMiddleware(logger)(svc)
		svc = InstrumentingMiddleware(pings)(svc)
	}
	return svc
}

var (
	// ErrRandomFailure is returned when the Service randomly fails.
	// It's meant to just demonstrate error handling.
	ErrRandomFailure = errors.New("random failure")
)

// NewBasicService returns a naïve, stateless implementation of Service.
func NewBasicService() Service {
	return basicService{}
}

type basicService struct{}

const (
	// This number says how often our Ping method should fail on purpose.
	// 1 = fail always, 0 = fail never, or a number in between.
	failureRate = 0.1
)

// TODO(nicolai): remember to seed random in main.go
// hackerservice.Ping uses rand
//rand.Seed(time.Now().UnixNano())

// Ping implements Service.
func (s basicService) Ping(_ context.Context) (string, error) {
	// TODO(nicolai): do this from main (all mains?)
	rand.Seed(time.Now().UnixNano())

	n := rand.Int63() % 100
	if n < (failureRate * 100) {
		return "", ErrRandomFailure
	}

	return "pong", nil
}
