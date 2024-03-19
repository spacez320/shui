package lib

import (
	"reflect"
	"testing"
	"time"

	"github.com/spacez320/cryptarch/pkg/storage"
)

func TestFilterResult(t *testing.T) {
	var (
		testResultValues         = make([]interface{}, 0)
		testFilteredResultValues = make([]interface{}, 0)
	)

	testTime, _ := time.Parse(time.ANSIC, time.Stamp)

	testResultValues = append(testResultValues, "foo")
	testResultValues = append(testResultValues, "bar")
	testResults := storage.Results{
		Labels: []string{"fizz"},
		Results: []storage.Result{
			{
				Time:   testTime,
				Value:  "foo bar",
				Values: testResultValues,
			},
		},
	}

	testFilteredResultValues = append(testFilteredResultValues, "foo")
	expected := storage.Result{
		Time:   testTime,
		Value:  "foo bar",
		Values: testFilteredResultValues,
	}

	got := FilterResult(testResults.Results[0], testResults.Labels, []string{"fizz"})

	// It successfully filtered result values.
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Got: %v Expected %v\n", got, expected)
	}
}
