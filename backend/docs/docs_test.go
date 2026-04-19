package docs

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSwaggerDoc_NoGlobalSecurity_ProtectedOpsRequireBearerAuth(t *testing.T) {
	raw := SwaggerInfo.ReadDoc()

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}

	if _, hasRootSecurity := doc["security"]; hasRootSecurity {
		t.Fatalf("root security must be absent; protected endpoints should declare BearerAuth explicitly, got %#v", doc["security"])
	}

	securityDefs, ok := doc["securityDefinitions"].(map[string]any)
	if !ok {
		t.Fatalf("securityDefinitions: got %#v", doc["securityDefinitions"])
	}
	if _, ok := securityDefs["BearerAuth"]; !ok {
		t.Fatalf("securityDefinitions: missing BearerAuth, got %#v", securityDefs)
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

			operationSecurity, hasOperationSecurity := op["security"].([]any)
			if hasOperationSecurity {
				for _, req := range operationSecurity {
					reqMap, ok := req.(map[string]any)
					if !ok {
						t.Fatalf("%s %s: security requirement shape invalid, got %#v", tc.method, tc.path, req)
					}
					if _, ok := reqMap[""]; ok {
						t.Fatalf("%s %s: found invalid empty security scheme key: %#v", tc.method, tc.path, reqMap)
					}
					if _, ok := reqMap["BearerAuth"]; ok {
						t.Fatalf("%s %s: public endpoint must not require BearerAuth, got %#v", tc.method, tc.path, reqMap)
					}
				}
			}

		})
	}

	for _, tc := range []struct {
		path   string
		method string
	}{
		{path: "/auth/providers/{provider}", method: "delete"},
		{path: "/auth/verify-email/send", method: "post"},
		{path: "/spend/redeem", method: "post"},
		{path: "/users/me", method: "get"},
		{path: "/users/me/addresses", method: "get"},
		{path: "/users/me/addresses", method: "post"},
		{path: "/users/me/addresses/{id}", method: "put"},
		{path: "/users/me/addresses/{id}", method: "delete"},
		{path: "/users/me/addresses/{id}/default", method: "put"},
		{path: "/users/me/points", method: "get"},
		{path: "/users/me/points/claim", method: "post"},
		{path: "/users/me/points/history", method: "get"},
		{path: "/users/me/providers", method: "get"},
		{path: "/users/me/tachi/balance", method: "get"},
	} {
		tc := tc
		t.Run(fmt.Sprintf("protected %s %s", tc.method, tc.path), func(t *testing.T) {
			pathItem, ok := paths[tc.path].(map[string]any)
			if !ok {
				t.Fatalf("%s: path item missing, got %#v", tc.path, paths[tc.path])
			}
			op, ok := pathItem[tc.method].(map[string]any)
			if !ok {
				t.Fatalf("%s %s: operation missing, got %#v", tc.method, tc.path, pathItem[tc.method])
			}
			securityReqs, ok := op["security"].([]any)
			if !ok || len(securityReqs) == 0 {
				t.Fatalf("%s %s: want non-empty security requirement, got %#v", tc.method, tc.path, op["security"])
			}
			firstReq, ok := securityReqs[0].(map[string]any)
			if !ok {
				t.Fatalf("%s %s: security requirement shape invalid, got %#v", tc.method, tc.path, securityReqs[0])
			}
			bearerScopes, ok := firstReq["BearerAuth"].([]any)
			if !ok || len(bearerScopes) != 0 {
				t.Fatalf("%s %s: want BearerAuth: [], got %#v", tc.method, tc.path, firstReq["BearerAuth"])
			}
			if _, ok := firstReq[""]; ok {
				t.Fatalf("%s %s: protected endpoint must not use invalid empty security scheme key: %#v", tc.method, tc.path, firstReq)
			}
		})
	}
}
