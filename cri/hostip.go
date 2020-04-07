package cri

import (
	"strings"
)

// Convert any host IP address in uri to nested CRI form if nested is true,
// or to legacy IPv4/IPv6 host IP address URI syntax otherwise.
func (f *Form) convHostIP(s string) string {


	// Scan past any URI scheme to find the body start and end
	i, j, _ := scanScheme(s)

	// Scan past the double-slash indicating the authority field
	if i+2 > j || s[i] != '/' || s[i+1] != '/' {
		return s // no authority field, so no IP address
	}
	i += 2

	// Scan past the userinfo field if there is one
	i = scanUserInfo(s, i)

	// Convert IPv4 and IPv6 address
	if j, addr := scanIP4(s, i); addr != "" { // a.b.c.d format
		if f.NestedIP {
			addr = "ip4[" + addr + "]" // to nested syntax
		}
		if !f.NestedIP || !f.Lazy { // convert address
			return s[:i] + addr + s[j:]
		}
	}
	if j, addr := scanIP6(s, i); addr != "" { // [xx:..:xx] format
		if f.NestedIP {
			addr = "ip6" + addr // bracketed already
		}
		if !f.NestedIP || !f.Lazy { // convert address
			return s[:i] + addr + s[j:]
		}
	}

	return s // no change
}

// Scan the authority part of a CRI for the end of a userinfo field.
// Returns the index of the end of that field, or 0 if no userinfo is found.
func scanUserInfo(s string, start int) (end int) {
	for i := start; i < len(s); i++ {
		switch {
		case s[i] == '@': // terminator for userinfo part
			return i + 1

		case isUnreserved(s[i]) || isPercEnc(s, i) ||
			isSubDelims(s[i]) || s[i] == ':':
			// scan past

		default:
			return start // invalid userinfo character
		}
	}
	return start
}

// Scan an IPv4 address in either legacy syntax like a.b.c.d,
// or in nested URI syntax like ip4[a.b.c.d].
func scanIP4(s string, start int) (end int, addr string) {

	// IPv4 address in legacy syntax
	end, addr = scanLegacyIP4(s, start)
	if addr != "" {
		return end, addr
	}

	// IPv4 address as a nested CRI
	i := start
	if len(s) < i+4 || strings.ToLower(s[i:i+4]) != "ip4[" {
		return 0, ""
	}
	le, addr := scanLegacyIP4(s, i+4)
	if addr == "" || le >= len(s) || s[le] != ']' {
		return 0, ""
	}
	end = le + 1
	return end, addr
}

func scanLegacyIP4(s string, start int) (end int, addr string) {
	i := start
	for k := 0; k < 4; k++ { // must have exactly 4 components
		if k > 0 {
			if len(s) <= i || s[i] != '.' {
				return 0, ""
			}
			i++
		}
		i = scanDec8(s, i)
		if i == 0 {
			return 0, ""
		}
	}
	return i, s[start:i]
}

// Scan a decimal octet in an IPv4 address
func scanDec8(s string, start int) (end int) {
	i := start
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i <= start || i > start+3 {
		return 0 // too few or too many decimal digits
	}
	return i
}

// Scan an IPv6 address in either legacy syntax like [xx:..:xx],
// or in nested URI syntax like ip6[xx:..:xx].
//
// Just searches for the closing square brackets
// while checking that intermediate characters are allowed
// in IPv6 addresses.
func scanIP6(s string, start int) (end int, addr string) {

	// IPv6 address in legacy syntax
	end, addr = scanLegacyIP6(s, start)
	if addr != "" {
		return end, addr
	}

	// IPv4 address as a nested CRI
	i := start
	if len(s) < i+4 || strings.ToLower(s[i:i+4]) != "ip6[" {
		return 0, ""
	}
	le, addr := scanLegacyIP6(s, i+3) // will scan the brackets
	if addr == "" {
		return 0, ""
	}
	end = le
	return end, addr
}

// Scan an IPv6 address in legacy syntax like [xx:..:xx].
// Just searches for the closing square brackets
// while checking that intermediate characters are allowed
// in IPv6 addresses.
func scanLegacyIP6(s string, start int) (end int, addr string) {

	// Scan the opening square bracket
	i := start
	if i >= len(s) || s[i] != '[' {
		return 0, ""
	}
	i++

	// Scan for closing square bracket
	for i < len(s) {
		switch s[i] {
		case ']': // found the close bracket
			end = i + 1
			return end, s[start:end]
		case ':', '.': // component separators
			// skip
		default:
			if !isHexDigit(s[i]) {
				return 0, "" // invalid character
			}
			// XXX support zone IDs
		}
		i++
	}
	return 0, "" // no close bracket
}

// Scan a 16-bit hexadecimal component in an IPv6 address
func scanHex16(s string, start int) (end int) {
	i := start
	for i < len(s) && isHexDigit(s[i]) {
		i++
	}
	if i <= start || i > start+4 {
		return 0 // too few or too many hex digits
	}
	return i
}
