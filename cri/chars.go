package cri

import ()

// Returns true if c is one of the unreserved characters defined in RFC 3986.
func isUnreserved(c byte) bool {
	switch c {
	case '-', '.', '_', '~':
		return true
	default:
		return isAlpha(c) || isDigit(c)
	}
}

// Returns true if c is an ASCII alphabetic character.
func isAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// Returns true if c is an ASCII digit.
func isDigit(c byte) bool {
	return (c >= '0' && c <= '9')
}

// Returns true if c is a hexadecimal digit.
func isHexDigit(c byte) bool {
	_, ok := getHexDigit(c)
	return ok
}

// Returns true if c is one of the gen-delims defined in RFC 3986.
func isGenDelims(c byte) bool {
	switch c {
	case ':', '/', '?', '#', '[', ']', '@':
		return true
	default:
		return false
	}
}

// Returns true if c is one of the sub-delims defined in RFC 3986.
func isSubDelims(c byte) bool {
	switch c {
	case '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=':
		return true
	default:
		return false
	}
}

// Returns true if a valid percent-encoded byte starts at index i in str.
func isPercEnc(str string, i int) bool {
	_, ok := getPercEnc(str, i)
	return ok
}

// Returns the percent-decoded byte at index i in str.
// Returns true in ok if successful and false otherwise.
func getPercEnc(str string, i int) (v byte, ok bool) {
	if len(str) < i+3 || str[i] != '%' {
		return 0, false // no percent-encoding here
	}
	vhi, okhi := getHexDigit(str[i+1])
	vlo, oklo := getHexDigit(str[i+2])
	if !okhi || !oklo {
		return 0, false // invalid percent-encoding
	}
	return vhi<<4 + vlo, true
}

// Returns the digit value of a hex digit c, or -1 if c is not a hex digit.
func getHexDigit(c byte) (v byte, ok bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}
