package lib

import (
	"reflect"
	"testing"
)

func TestFilterSlice(t *testing.T) {
	expected := []string{"foo", "bar"}
	got := FilterSlice([]string{"fizz", "foo", "bar", "bizz"}, []int{1, 2})

	// It filters a slice with provided indexes.
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Got: %v Expected %v\n", got, expected)
	}

	expected = []string{"foo", "bar"}
	got = FilterSlice([]string{"foo", "bar"}, []int{})

	// It returns the original slice if indexes is empty.
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Got: %v Expected %v\n", got, expected)
	}
}
