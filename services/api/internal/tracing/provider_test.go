package tracing

import (
	"context"
	"strings"
	"testing"

	"github.com/tachigo/tachigo/internal/config"
)

func TestConfigureProviderDisabledIsNoop(t *testing.T) {
	shutdown, err := ConfigureProvider(context.Background(), config.TracingConfig{})
	if err != nil {
		t.Fatalf("ConfigureProvider() error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestConfigureProviderReturnsValidationError(t *testing.T) {
	_, err := ConfigureProvider(context.Background(), config.TracingConfig{
		Enabled:     true,
		ServiceName: "tachigo-api",
		Environment: "staging",
		SampleRatio: 0.10,
	})
	if err == nil {
		t.Fatal("ConfigureProvider() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") {
		t.Fatalf("ConfigureProvider() error = %q", err.Error())
	}
}

func TestConfigureProviderAcceptsValidLocalOTLPConfig(t *testing.T) {
	shutdown, err := ConfigureProvider(context.Background(), config.TracingConfig{
		Enabled:            true,
		ServiceName:        "tachigo-api",
		Environment:        "development",
		SampleRatio:        0.10,
		OTLPTracesEndpoint: "http://localhost:4318/v1/traces",
		OTLPInsecure:       true,
	})
	if err != nil {
		t.Fatalf("ConfigureProvider() error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}
