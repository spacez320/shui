package storage

import (
	"testing"
)

func TestQueryToPromName(t *testing.T) {
	tests := map[string]string{
		"test":          "test",
		"foo__bar":      "foo_bar",
		"!foo1?:bar2!:": "foo1_bar2",
	}

	// It gets a result matching the time.
	for input, expected := range tests {
		if got := queryToPromMetricName(input); got != expected {
			t.Errorf("Got: %v Expected: %v\n", got, expected)
		}
	}
}
