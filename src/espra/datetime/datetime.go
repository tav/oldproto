// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package datetime

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

var timePrefix string

const (
	PrefixFactor = 1 // Max: 256
)

type Timestamp string

func (ts Timestamp) Time() (time.Time, error) {
	i, err := strconv.ParseInt(string(ts)[1:], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, i), nil
}

// From converts a Time object into a Timestamp string.
func From(t time.Time) Timestamp {
	return Timestamp(fmt.Sprintf("%s%019d\n", timePrefix, t.UnixNano()))
}

// Now returns the current UTC time as a Timestamp string.
func Now() Timestamp {
	return From(time.Now())
}

// UTC returns the current time in UTC.
func UTC() time.Time {
	return time.Now().UTC()
}

func init() {
	rand.Seed(time.Now().UnixNano())
	timePrefix = string(rand.Int() % PrefixFactor)
}
