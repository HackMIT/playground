package utils

import "unicode"

func IsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			// TODO: Send error packet
			return false
		}
	}

	return true
}
