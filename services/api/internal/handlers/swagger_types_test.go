package handlers_test

import (
	"reflect"
	"testing"

	"github.com/tachigo/tachigo/internal/handlers"
)

func TestPointsHistoryItem_SwaggerTagsExposeContract(t *testing.T) {
	itemType := reflect.TypeOf(handlers.PointsHistoryItem{})

	typeField, ok := itemType.FieldByName("Type")
	if !ok {
		t.Fatal("PointsHistoryItem.Type field not found")
	}
	if got := typeField.Tag.Get("enums"); got != "earn,spend" {
		t.Fatalf("Type enums tag: want earn,spend, got %q", got)
	}

	createdAtField, ok := itemType.FieldByName("CreatedAt")
	if !ok {
		t.Fatal("PointsHistoryItem.CreatedAt field not found")
	}
	if got := createdAtField.Tag.Get("format"); got != "date-time" {
		t.Fatalf("CreatedAt format tag: want date-time, got %q", got)
	}
}
