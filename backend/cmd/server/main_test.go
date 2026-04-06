package main

import (
	"strings"
	"testing"
)

func TestInitializeUserRoleEnumFreshDatabase(t *testing.T) {
	var statements []string

	err := initializeUserRoleEnum(func(query string) error {
		statements = append(statements, normalizeSQL(query))
		return nil
	})
	if err != nil {
		t.Fatalf("initializeUserRoleEnum returned error: %v", err)
	}

	if len(statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(statements))
	}
	if !strings.Contains(statements[0], "CREATE TYPE user_role AS ENUM ('viewer', 'streamer', 'agency', 'admin')") {
		t.Fatalf("first statement should create enum, got %q", statements[0])
	}
	if statements[1] != "ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'agency'" {
		t.Fatalf("second statement should add agency enum value, got %q", statements[1])
	}
}

func TestInitializeUserRoleEnumExistingDatabaseAddsAgencySeparately(t *testing.T) {
	var statements []string

	err := initializeUserRoleEnum(func(query string) error {
		normalized := normalizeSQL(query)
		statements = append(statements, normalized)
		if strings.Contains(normalized, "DO $$ BEGIN") && strings.Contains(normalized, "CREATE TYPE user_role") {
			return nil
		}
		return nil
	})
	if err != nil {
		t.Fatalf("initializeUserRoleEnum returned error: %v", err)
	}

	if len(statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(statements))
	}
	if strings.Contains(statements[0], "ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'agency'") {
		t.Fatalf("ALTER TYPE must not be inside the DO block: %q", statements[0])
	}
	if statements[1] != "ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'agency'" {
		t.Fatalf("expected standalone ALTER TYPE statement, got %q", statements[1])
	}
}

func normalizeSQL(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
