package testutil

import (
	"io"
	"net/http"
	"strings"
)

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func NewHTTPClient(rt RoundTripperFunc) *http.Client {
	return &http.Client{Transport: rt}
}

func NewStringResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
