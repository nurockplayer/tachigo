package docs

import (
	"encoding/json"
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

func TestSwaggerArtifacts_DoNotDocumentRootHealthEndpoints(t *testing.T) {
	path := filepath.Join("swagger.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	for _, endpoint := range []string{"/health", "/readyz"} {
		if swaggerDocumentContainsPath(t, raw, endpoint) {
			t.Fatalf("%s documents root operational endpoint %s", path, endpoint)
		}
	}
}

func TestSwaggerDocumentContainsPathIgnoresNonPathText(t *testing.T) {
	raw := []byte(`{"info":{"x-root-probes":["/health"]},"paths":{"/api/v1/users":{}}}`)

	if swaggerDocumentContainsPath(t, raw, "/health") {
		t.Fatal("non-path text should not be treated as a documented Swagger path")
	}
}

func swaggerDocumentContainsPath(t *testing.T, raw []byte, endpoint string) bool {
	t.Helper()

	var doc struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal swagger document: %v", err)
	}
	_, ok := doc.Paths[endpoint]
	return ok
}
