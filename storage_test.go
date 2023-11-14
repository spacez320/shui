package main

import (
	"testing"
	"time"
)

var testTime time.Time

// Build a test storage.
func testStorage() Results {
	testTime, _ := time.Parse(time.ANSIC, time.Stamp)

	return Results{
		result{
			Time:  testTime,
			Value: "foo",
		},
		result{
			Time:  testTime.Add(time.Second * 30),
			Value: "bar",
		},
	}
}

func TestGet(t *testing.T) {
	results := testStorage()

	got := results.Get(testTime)
	expected := results.GetI(0)

	if got != expected {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}
}

func TestGetRange(t *testing.T) {
	return
}

func TestPut(t *testing.T) {
	return
}

func TestShow(t *testing.T) {
	return
}
