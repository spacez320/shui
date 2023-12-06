//
// General error management.

package lib

import "golang.org/x/exp/slog"

// General error manager.
func e(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}
