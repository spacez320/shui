//
// Reader indexes allow for tracking the last read result from a results series.

package storage

import "github.com/google/uuid"

// Reader indexes control where a consumer has last read a result.
type ReaderIndex int

// Client id for reader indexes.
type ReaderIndexClientId uuid.UUID

// Decrement a reader index, to re-read the last read.
func (i *ReaderIndex) Dec() {
	(*i)--
}

// Incremement a reader index, likely after a read.
func (i *ReaderIndex) Inc() {
	(*i)++
}

// Sets a redaer index to a specified value.
func (i *ReaderIndex) Set(newI int) {
	*i = ReaderIndex(newI)
}
