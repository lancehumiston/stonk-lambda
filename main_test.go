package main

import "testing"

func TestUnique_Duplicates_ReturnsUniqueItems(t *testing.T) {
	items := []string{
		"a",
		"b",
		"c",
		"b",
		"c",
		"d",
	}
	expected := []string{
		"a",
		"b",
		"c",
		"d",
	}

	result := unique(items)

	if !IsEqual(result, expected) {
		t.Fatalf("Failed expected:%v actual:%v", expected, result)
	}
}

func IsEqual(a, b []string) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
