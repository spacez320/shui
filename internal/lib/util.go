//
// General utilities.

package lib

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
