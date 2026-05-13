package services

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func installDBContextProbe(t *testing.T, db *gorm.DB, key, want any) func() int {
	t.Helper()

	var seen int
	name := "test:db_context:" + uuid.NewString()
	probe := func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Context != nil && tx.Statement.Context.Value(key) == want {
			seen++
		}
	}

	if err := db.Callback().Create().Before("gorm:create").Register(name+":create", probe); err != nil {
		t.Fatalf("register create context probe: %v", err)
	}
	if err := db.Callback().Query().Before("gorm:query").Register(name+":query", probe); err != nil {
		t.Fatalf("register query context probe: %v", err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register(name+":update", probe); err != nil {
		t.Fatalf("register update context probe: %v", err)
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register(name+":raw", probe); err != nil {
		t.Fatalf("register raw context probe: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(name + ":create")
		_ = db.Callback().Query().Remove(name + ":query")
		_ = db.Callback().Update().Remove(name + ":update")
		_ = db.Callback().Raw().Remove(name + ":raw")
	})

	return func() int {
		return seen
	}
}
