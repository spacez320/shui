package main

import (
	"testing"
	"time"
)

// General time stamp to use for testing storage operations.
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

	// It gets a result matching the time.
	got := results.Get(testTime)
	expected := results[0]

	if got != expected {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}

	// It gets no results if a time is far advanced.
	got = results.Get(testTime.Add(time.Second * 60))
	expected = result{}

	if got != expected {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}

	// It gets the first result if the time is before the first result.
	got = results.Get(testTime.Add(-time.Second * 60))
	expected = results[0]

	if got != expected {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}

	// It gets the second result if the time is between the first and second
	// results.
	got = results.Get(testTime.Add(time.Second * 15))
	expected = results[1]

	if got != expected {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}
}

func TestGetRange(t *testing.T) {
	results := testStorage()

	// It gets results for exact matches on a time range.
	got := results.GetRange(testTime, testTime.Add(time.Second*30))
	expected := results

	for i, result := range got {
		if result != expected[i] {
			t.Errorf("Got: %v Expected: %v\n", got, expected)
			break
		}
	}

	// It gets results for extended matches on a time range.
	got = results.GetRange(testTime.Add(-time.Second*30), testTime.Add(time.Second*60))

	for i, result := range got {
		if result != expected[i] {
			t.Errorf("Got: %v Expected: %v\n", got, expected)
			break
		}
	}

	// It returns a single result if the time range is restricted.
	got = results.GetRange(testTime, testTime)

	if len(got) != 1 && got[0] != expected[0] {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}
}

func TestPut(t *testing.T) {
	results := testStorage()

	// It successfully appends a result.
	results.Put("fizz")

	if len(results) != 3 && results[2].Value != "fizz" {
		t.Errorf("Got: %v\n", results)
	}
}
