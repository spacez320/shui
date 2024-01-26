package lib

import "testing"

func TestByteConv(t *testing.T) {
	expected := 1.23
	got, _ := byteConv(1230, "kilobyte")
	if got != expected {
		t.Errorf("Got: %v Expected %v\n", got, expected)
	}

	got, _ = byteConv(1230000, "megabyte")
	if got != expected {
		t.Errorf("Got: %v Expected %v\n", got, expected)
	}

	got, _ = byteConv(1230000000, "gigabyte")
	if got != expected {
		t.Errorf("Got: %v Expected %v\n", got, expected)
	}
}
