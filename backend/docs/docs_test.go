package docs

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSwaggerDoc_UsesGlobalBearerAuthAndPublicOverrides(t *testing.T) {
	raw := SwaggerInfo.ReadDoc()

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}

	security, ok := doc["security"].([]any)
	if !ok || len(security) != 1 {
		t.Fatalf("root security: want one BearerAuth requirement, got %#v", doc["security"])
	}

	firstReq, ok := security[0].(map[string]any)
	if !ok {
		t.Fatalf("root security requirement shape: got %#v", security[0])
	}
	if _, ok := firstReq["BearerAuth"]; !ok {
		t.Fatalf("root security: want BearerAuth, got %#v", firstReq)
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths: got %#v", doc["paths"])
	}

	for _, tc := range []struct {
		path   string
		method string
	}{
		{path: "/auth/google", method: "get"},
		{path: "/auth/google/callback", method: "get"},
		{path: "/auth/twitch", method: "get"},
		{path: "/auth/twitch/callback", method: "get"},
		{path: "/auth/login", method: "post"},
		{path: "/auth/register", method: "post"},
		{path: "/auth/refresh", method: "post"},
		{path: "/auth/logout", method: "post"},
		{path: "/auth/web3/nonce", method: "post"},
		{path: "/auth/web3/verify", method: "post"},
		{path: "/auth/verify-email/confirm", method: "post"},
		{path: "/auth/forgot-password", method: "post"},
		{path: "/auth/reset-password", method: "post"},
		{path: "/extension/auth/login", method: "post"},
		{path: "/extension/bits/complete", method: "post"},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			pathItem, ok := paths[tc.path].(map[string]any)
			if !ok {
				t.Fatalf("%s: path item missing, got %#v", tc.path, paths[tc.path])
			}
			op, ok := pathItem[tc.method].(map[string]any)
			if !ok {
				t.Fatalf("%s %s: operation missing, got %#v", tc.method, tc.path, pathItem[tc.method])
			}
			override, ok := op["security"].([]any)
			if !ok {
				t.Fatalf("%s %s: security override missing, got %#v", tc.method, tc.path, op["security"])
			}
			if len(override) != 0 {
				t.Fatalf("%s %s: want empty security override, got %#v", tc.method, tc.path, override)
			}
		})
	}
}
