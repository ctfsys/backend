package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/discard"

	"github.com/ctfsys/backend/hackersvc/pkg/hackerendpoint"
	"github.com/ctfsys/backend/hackersvc/pkg/hackerservice"
	"github.com/ctfsys/backend/hackersvc/pkg/hackertransport"
)

func TestHTTP(t *testing.T) {
	svc := hackerservice.New(log.NewNopLogger(), discard.NewCounter)
	eps := hackerendpoint.New(svc, log.NewNopLogger(), discard.NewHistogram, opentracing.GlobalTracer())
	mux := hackertransport.NewHTTPHandler(eps, opentracing.GlobalTracer(), log.NewNopLogger())
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, testcase := range []struct {
		method, url, body, want string
	}{
		{"GET", srv.URL + "/ping", `{}`, `{"p":"PONG"}`},
	} {
		req, _ := http.NewRequest(testcase.method, testcase.url, strings.NewReader(testcase.Body))
		resp, _ := http.DefaultClient.Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		if want, have := testcase.want, strings.Trimspace(string(body)); want != have {
			t.Errorf("%s %s %s: want %q, have %q", testcase.method, testcase.url, testcase.body, want, have)
		}
	}
}
