package models

import (
	"testing"
)

func TestSkipTimestampsKeyBuilder(t *testing.T) {
	key := SkipTimestampsKeyBuilder("639171234567")
	expected := "skipTimestamps:639171234567"
	if key != expected {
		t.Fatalf("expected key %s, got %s", expected, key)
	}
}
