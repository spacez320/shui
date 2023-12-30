package storage

import (
	"reflect"
	"testing"
	"time"
)

// General time stamp to use for testing storage operations.
var testTime time.Time

// Build a test storage.
func testStorage() Results {
	testTime, _ := time.Parse(time.ANSIC, time.Stamp)

	return Results{
		Labels: []string{"foo", "bar"},
		Results: []Result{
			Result{
				Time:   testTime,
				Value:  "foo",
				Values: nil,
			},
			Result{
				Time:   testTime.Add(time.Second * 30),
				Value:  "bar",
				Values: nil,
			},
		},
	}
}

func TestGet(t *testing.T) {
	results := testStorage()

	// It gets a result matching the time.
	got := results.Get(testTime)
	expected := results.Results[0]
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}

	// It gets no results if a time does not match.
	got = results.Get(testTime.Add(time.Second * 15))
	expected = Result{}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}
}

func TestGetRange(t *testing.T) {
	results := testStorage()

	// It gets results for exact matches on a time range.
	got := results.GetRange(testTime, testTime.Add(time.Second*30))
	expected := results
	for i, result := range got {
		if !reflect.DeepEqual(result, expected.Results[i]) {
			t.Errorf("Got: %v Expected: %v\n", got, expected)
			break
		}
	}

	// It gets results for extended matches on a time range.
	got = results.GetRange(testTime.Add(-time.Second*30), testTime.Add(time.Second*60))
	for i, result := range got {
		if !reflect.DeepEqual(result, expected.Results[i]) {
			t.Errorf("Got: %v Expected: %v\n", got, expected)
			break
		}
	}

	// It returns a single result if the time range is restricted.
	got = results.GetRange(testTime, testTime)
	if len(got) != 1 || !reflect.DeepEqual(got[0], expected.Results[0]) {
		t.Errorf("Got: %v Expected: %v\n", got, expected)
	}
}

func TestPut(t *testing.T) {
	results := testStorage()

	// It successfully appends a result.
	results.Put("fizz")
	if len(results.Results) != 3 && results.Results[2].Value != "fizz" {
		t.Errorf("Got: %v\n", results)
	}
}

func TestPutC(t *testing.T) {
	results := testStorage()

	// It successfully appends a compound result.
	results.Put("fizz", "fizz", 3)
	expected := make([]interface{}, 0)
	expected = append(expected, "fizz")
	expected = append(expected, 3)
	for i, result := range results.Results[2].Values {
		if result != expected[i] {
			t.Errorf("Got: %v\n", results)
		}
	}
}
