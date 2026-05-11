package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwaggerArtifacts_NoInvalidEmptySecurityScheme(t *testing.T) {
	path := filepath.Join("swagger.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if strings.Contains(string(raw), `"": []`) {
		t.Fatalf("%s contains invalid Swagger 2.0 security requirement \"\": []", path)
	}
}

func TestSwaggerArtifacts_IncludeRootHealthEndpoints(t *testing.T) {
	path := filepath.Join("swagger.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	for _, endpoint := range []string{`"/health"`, `"/readyz"`} {
		if !strings.Contains(string(raw), endpoint) {
			t.Fatalf("%s missing root endpoint %s", path, endpoint)
		}
	}
}
