// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package ident

import (
	"strings"
)

func normalise(ident string, skipFirst bool) (string, bool) {
	str := []byte{}
	idx := 0
	if skipFirst {
		if len(ident) <= 2 {
			return "", false
		}
		str = append(str, ident[0])
		idx = 1
	} else {
		if len(ident) <= 1 {
			return "", false
		}
	}
	ident = strings.ToLower(ident)
	if ident[len(ident)-1] == '-' {
		return "", false
	}
	char := ident[idx]
	if char < 'a' || char > 'z' {
		return "", false
	}
	str = append(str, char)
	prevDash := -1
	for i := idx + 1; i < len(ident); i++ {
		char := ident[i]
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			str = append(str, char)
		} else if char == '-' {
			if i == (prevDash + 1) {
				return "", false
			}
			prevDash = i
			str = append(str, char)
		} else {
			return "", false
		}
	}
	return string(str), true
}

func Username(ident string) (string, bool) {
	return normalise(ident, false)
}

func UserRef(ident string) (string, bool) {
	if len(ident) <= 2 {
		return "", false
	}
	if ident[0] != '+' {
		return "", false
	}
	return normalise(ident, true)
}

func Ref(ident string) (string, bool) {
	if ident[0] != '#' || ident[0] != '+' {
		return "", false
	}
	return normalise(ident, true)
}
