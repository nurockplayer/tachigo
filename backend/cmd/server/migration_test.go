package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigration008AddsAgencyUserForeignKeyConstraint(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "008_streamers_agency.sql")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(body)
	if !strings.Contains(sql, "fk_streamers_agency_user_id") {
		t.Fatalf("migration must create named fk_streamers_agency_user_id constraint")
	}
	if !strings.Contains(sql, "FOREIGN KEY (agency_user_id) REFERENCES users(id)") {
		t.Fatalf("migration must add foreign key on streamers.agency_user_id")
	}
}
