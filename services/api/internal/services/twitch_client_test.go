package services

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func googleUserTestContext(status int, body string) context.Context {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://www.googleapis.com/oauth2/v3/userinfo" {
				return nil, http.ErrNotSupported
			}
			return &http.Response{
				StatusCode: status,
				Status:     http.StatusText(status),
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
	return context.WithValue(context.Background(), oauth2.HTTPClient, client)
}

func TestFetchGoogleUser_Non2xxStatusReturnsError(t *testing.T) {
	ctx := googleUserTestContext(http.StatusUnauthorized, `{"sub":"google-123","email":"viewer@example.com"}`)

	_, err := fetchGoogleUser(ctx, &oauth2.Config{}, &oauth2.Token{AccessToken: "token"})
	if err == nil {
		t.Fatal("want error for non-2xx Google userinfo response, got nil")
	}
	if !strings.Contains(err.Error(), "google API returned 401") {
		t.Fatalf("want status error, got %v", err)
	}
}

func TestFetchGoogleUser_MissingSubReturnsError(t *testing.T) {
	ctx := googleUserTestContext(http.StatusOK, `{"email":"viewer@example.com","name":"Viewer"}`)

	_, err := fetchGoogleUser(ctx, &oauth2.Config{}, &oauth2.Token{AccessToken: "token"})
	if err == nil {
		t.Fatal("want error for missing Google sub, got nil")
	}
	if !strings.Contains(err.Error(), "missing sub") {
		t.Fatalf("want missing sub error, got %v", err)
	}
}

func TestFetchGoogleUser_MissingEmailReturnsError(t *testing.T) {
	ctx := googleUserTestContext(http.StatusOK, `{"sub":"google-123","name":"Viewer"}`)

	_, err := fetchGoogleUser(ctx, &oauth2.Config{}, &oauth2.Token{AccessToken: "token"})
	if err == nil {
		t.Fatal("want error for missing Google email, got nil")
	}
	if !strings.Contains(err.Error(), "missing email") {
		t.Fatalf("want missing email error, got %v", err)
	}
}
