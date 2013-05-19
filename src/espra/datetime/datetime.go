// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package datetime

import (
	"time"
)

var timePrefix byte

const (
	Epoch = 0
)

type Timestamp string

func (ts Timestamp) Time() time.Time {
	ts = ts[1:]
	return time.Now()
}

// From converts a Time object into a Timestamp string.
func From(t time.Time) Timestamp {
	return ""
}

// Now returns the current UTC time as a Timestamp string.
func Now() Timestamp {
	return From(time.Now().UTC())
}

// UTC returns the current time in UTC.
func UTC() time.Time {
	return time.Now().UTC()
}

func init() {
	timePrefix = 0
}
