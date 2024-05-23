//
// General error management.

package lib

import (
	"log/slog"
)

// General error manager.
func e(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}
