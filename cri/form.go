// This package contains functions to convert
// composable resource identifiers (CRIs)
// to or from conventional URIs.
//
// For background information on CRIs,
// see the draft blog post at https://bford.info/draft/cri/
// (warning: this is a temporary link that will change).
//
// For now, the recommended way to parse a CRI structurally
// is to convert it to a URI then parse it using net.url.Parse
// (see https://golang.org/pkg/net/url/).
//
// This is early, incomplete, experimental code with many limitations,
// including not yet dealing with internationalized IRIs.
//
package cri

import (
	"errors"
)

// Check
//func Check(cri string) error {
//}

// Form describes a resource identifier's form in terms of extensions allowed.
type Form struct {
	Unicode  bool // Allow internationalized UCS characters
	Brackets bool // Use square brackets to delimit body
	NestedIP bool // Host IP addresses in nested CRI syntax
	Lazy     bool // Minimize changes, don't raise expressiveness

	mayGrow struct{} // Private field to guard extensibility
}

// Configuration for legacy ASCII-only URIs (RFC 3986)
var URI = &Form{}

// Configuration for internationalized resource identifiers (RFC 3987)
var IRI = &Form{Unicode: true}

// Configuration for composable resource identifiers
var CRI = &Form{Unicode: true, Brackets: true, NestedIP: true}

// Check whether resource identifier ri conforms to this Form.
// Returns nil if so, and otherwise an error indicating one reason it doesn't.
//
// XXX this function does not (yet) attempt to check syntax exhaustively.
//
func (f *Form) Check(ri string) error {

	// Check characters allowed
	for _, r := range ri {
		if r >= 128 && !f.Unicode {
			return errNoUnicode
		}
	}

	// XXX structural checks

	return nil
}

// Attempt to convert resource identifier ri to the designated Form.
//
// This function attempts to be permissive in what it accepts,
// and does not deeply parse or validate the input resource identifier.
//
func (f *Form) From(ri string) (new string, err error) {

	// Percent-encode Unicode characters if not allowed in target Form
	if !f.Unicode {
		ri, err = f.fromUnicode(ri)
		if err != nil {
			return "", err
		}
	}

	if !f.Lazy {
		// XXX de-percent-encode characters that we're allowed to in this Form
	}

	// Break out the scheme name and locate the RI's body
	bodyStart, bodyEnd, delim := scanScheme(ri)

	// Convert from bracketed to colon-delimited form if needed
	if delim == '[' && !f.Brackets {
		ri = ri[:bodyStart-1] + ":" + ri[bodyStart:bodyEnd]
	}

	// Convert from colon-delimited to bracketed if appropriate
	if delim == ':' && f.Brackets && !f.Lazy {
		ri = ri[:bodyStart-1] + "[" + ri[bodyStart:bodyEnd] + "]"
	}

	// XXX deal with embedded brackets appropriately

	// Convert host IP addresses as appropriate
	ri = f.convHostIP(ri)

	return ri, nil
}

// Percent-encode any Unicode characters in RI
func (f *Form) fromUnicode(ri string) (string, error) {

	// XXX implement
	return ri, nil
}

// Scan for the scheme name in a resource identifier
// On success returns the start and end of the CRI's body
// and the ':' or '[' delimiter separating it from the scheme name.
// Returns 0, len(s), 0 if the input s has no valid scheme name.
//
func scanScheme(s string) (start, end int, delim byte) {
	start, end = 0, len(s)

	// Scheme names must start with an alphabetic ASCII character.
	if end == 0 || !isAlpha(s[0]) {
		return start, end, 0
	}

	// Scan the rest of the scheme name
	for i := 1; i < end; i++ {
		c := s[i]
		switch {
		case isAlpha(c) || isDigit(c) ||
			c == '+' || c == '-' || c == '.':
			// scan past valid scheme name characters

		case c == ':':
			return i + 1, end, c // colon-delimited RI

		case c == '[':
			// Find matching close bracket
			if s[end-1] != ']' {
				return start, end, 0 // no close bracket
			}
			return i + 1, end - 1, c

		default: // invalid scheme character
			return start, end, 0

		}
	}
	return start, end, 0 // no scheme name found
}

// Returns str after decoding any percent-encoded unreserved characters.
func decodeUnreserved(s string) string {
	for i := 0; i < len(s); i++ {
		if c, ok := getPercEnc(s, i); ok && isUnreserved(c) {
			s = s[:i] + string(c) + s[i+3:]
		}
	}
	return s
}

var errNoUnicode = errors.New("Unicode characters not allowed")
