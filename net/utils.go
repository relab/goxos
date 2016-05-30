package net

import (
	"strings"
)

func IsSocketClosed(err error) bool {
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}

	return false
}
