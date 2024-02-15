//
// Storage specific errors.

package storage

import (
	"fmt"
)

// Error indicating a number is expected but was not provided.
type NaNError struct {
	// Value attempted to be used
	Value interface{}
}

func (e *NaNError) Error() string {
	return fmt.Sprintf("Attempted to use non-numeric value in numerical context. Value: %v", e.Value)
}
