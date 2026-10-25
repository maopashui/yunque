package docs

import "testing"

func TestOrderKeyBetween(t *testing.T) {
	first, err := OrderKeyBetween("", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := OrderKeyBetween(first, "")
	if err != nil {
		t.Fatal(err)
	}
	if !(first < second) {
		t.Fatalf("expected %q < %q", first, second)
	}
	mid, err := OrderKeyBetween(first, second)
	if err != nil {
		t.Fatal(err)
	}
	if !(first < mid && mid < second) {
		t.Fatalf("expected %q < %q < %q", first, mid, second)
	}
}
