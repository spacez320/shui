//
// General utilities.

package lib

import "golang.org/x/exp/slices"

// Gets the next element in a slice, with wrap-around if selecting from the
// last element.
func GetNextSliceRing[T comparable](in []T, current T) T {
	return in[(slices.Index(in, current)+1)%len(in)]
}

// Pick items from an arbitrary slice according to provided indexes. If indexes
// is empty, it will just return the original slice.
func FilterSlice[T interface{}](in []T, indexes []int) (out []T) {
	if len(indexes) == 0 {
		out = in
	} else {
		for _, index := range indexes {
			out = append(out, in[index])
		}
	}

	return
}

// Gives a new percentage based on globalRelativePerc after reducing totality
// by limiting Perc.
//
// For example, given a three-way percentage split of 80/10/10, this function
// will return 50 if given the arguments 80 and 10.
func RelativePerc(limitingPerc, globalRelativePerc int) int {
	return (100 * globalRelativePerc) / (100 - limitingPerc)
}
