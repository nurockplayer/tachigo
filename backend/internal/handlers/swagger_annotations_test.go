package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestSwaggerAnnotations_NoEmptySecurityDirective(t *testing.T) {
	t.Helper()

	emptySecurity := regexp.MustCompile(`(?m)^//\s*@Security\s*$`)

	for _, rel := range []string{
		"auth_handler.go",
		"email_auth_handler.go",
		"extension_handler.go",
	} {
		path := filepath.Join(rel)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}

		if emptySecurity.Match(b) {
			t.Fatalf("%s contains empty @Security directive; remove it from public endpoints", rel)
		}
	}
}
