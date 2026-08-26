package validation

import "testing"

func TestRequired(t *testing.T) {
	if err := Required("name", "value"); err != nil {
		t.Fatalf("Required returned error for a value: %v", err)
	}
	if err := Required("name", "  "); err == nil {
		t.Fatal("Required returned nil for whitespace")
	}
}

func TestIntegerRange(t *testing.T) {
	for _, value := range []int{0, 5, 10} {
		if err := IntegerRange("mood", value, 0, 10); err != nil {
			t.Fatalf("IntegerRange(%d) returned error: %v", value, err)
		}
	}
	if err := IntegerRange("mood", 11, 0, 10); err == nil {
		t.Fatal("IntegerRange returned nil for an out-of-range value")
	}
}
