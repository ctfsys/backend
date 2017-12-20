package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

func TestPactHackersvcPing(t *testing.T) {
	if os.Getenv("WRITE_PACTS") == "" {
		t.Skip("skipping Pact contracts; set WRITE_PACTS environment variable to enable")
	}

	pact := dsl.Pact{
		Port:     6666,
		Consumer: "hackersvc",
		Provider: "testsvc", // TODO(nicolai): how do we actually do this?
	}
	defer pact.Teardown()

	pact.AddInteraction().
		UponReceiving("testsvc ping").
		WithRequest(dsl.Request{
			Headers: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Method:  "POST",
			Path:    "/ping",
			Body:    `"{}"`,
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: map[string]string{"Content-Type": "application/json; charset=urf-8"},
			Body:    `{"p":"PONG"}`,
		})

	if err := pact.Verify(func() error {
		u := fmt.Sprintf("http://localhost:%d/ping", pact.Server.Port)
		req, err := http.NewRequest("POST", u, strings.NewReader(`{}`))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		if _, err = http.DefaultClient.Do(req); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	pact.WritePact()
}
