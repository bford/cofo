// Composable Text Syntax (CTS).
// 
// See XXX for more information.
package cts

import (
	"io"
	"bufio"
	"errors"
	"strings"
)

// Brackets defines a punctuation configuration for a CTS encoder/decoder
// in terms of the set of open/close punctuation characters that are sensitive.
// The string must be an even-length,
// in which each two characters defines a matching open-close pair.
//
// These pairs should generally be limited to pairs of
// Open Punctuation (Ps) and Close Punctuation (Pe) characters
// as defined by Unicode's character properties. For reference see:
// 
//	https://www.compart.com/en/unicode/category/Ps
//	https://www.compart.com/en/unicode/category/Pe
//
// Although it is possible, it is not recommended to 
// use the ASCII '<' and '>' characters as sensitive brackets,
// because these are formally defined as and most commonly used as
// mathematical less-than and greater-than signs, respectively.
// Unicode defines separate open and close angle-brackets, '〈' and '〉'.
//
type Brackets string

// Minimal bracket configuration in which
// only the square brackets [] are sensitive.
const SquareBrackets Brackets = "[]"

// Bracket configuration in which all the ASCII open/close
// punctuation character pairs are sensitive.
const AsciiBrackets Brackets = "()[]{}"

// Bracket configuration in which all the cleanly-matched Unicode open/close
// punctuation character pairs are sensitive.
// const AllBrackets Brackets = AsciiBrackets +
// 	"\u0F3A\u0F3B" +	// Tibetan Mark Gug Rtags Gyon, Gyas
// 	"\u0F3C\u0F3D" +	// Tibetan Mark Ang Khang Gyon, Gyas
// 	"\u169B\u169C" +	// Ogham Feather Mark, Reversed Feather Mark
// 	"" ...
// 
// (if someone has time to complete this...


type bracket struct {
	other rune		// other matching bracket
	close bool		// false if open bracket, true if close bracket
}

type pairs map[rune]bracket	// Map from runes to matching partner info

// Convert a bracket config string into easier-to-use correspondence maps.
func newPairs(b Brackets) pairs {

	if b == "" {
		b = SquareBrackets	// default bracket configuration
	}

	// make sure it contains an even-length number of runes
	r := []rune(b)
	if (len(r) & 1) != 0 {
		panic("Brackets string must be even length")
	}

	// Build the matched pair maps
	p := make(pairs)
	for i := 0; i < len(r); i += 2 {
		op := r[i]
		cl := r[i+1]
		p[op] = bracket{cl, false}
		p[cl] = bracket{op, true}
	}
	return p
}


// Configuration options for the BTS encoder/decoder.
type Config struct {
	Tolerant	bool	// true to tolerate recoverable syntax errors

	Brackets	Brackets // which character pairs are sensitive

	// Function to handle decoding errors as they occur.
	// If this function returns non-nil, decoding stops with that error.
	// But this function can return nil to (try to) continue decoding.
	// If this function is nil, the default is to stop at the first error.
	HandleError	func(error) error
}

// A Decoder reads structured CTS values from an input stream.
type Decoder struct {
	r *bufio.Reader
	p pairs
	h func(error) error
	b strings.Builder
	e error
}

// Create a new Decoder that reads UTF-8 encoded input text from r.
func (c *Config) NewDecoder(r io.Reader) *Decoder {

	h := c.HandleError
	if h == nil {
		h = func(e error) error { return e } // default error handler
	}

	return &Decoder{r: bufio.NewReader(r),
			p: newPairs(c.Brackets),
			h: h}
}

func (dec *Decoder) toBracket(close rune) (rune, rune, error) {
	for {
		rune, _, err := dec.r.ReadRune()
		if err != nil {
			return 0, 0, err // we have to stop at EOF or I/O error
		}
		if close != 0 && rune == close { // found closer we wanted
			return 0, 0, nil
		}
		if br, ok := dec.p[rune]; ok {	// found a bracket?
			if close == 0 && !br.close { // found open bracket
				return rune, br.other, nil

			} else if close == 0 { // found close looking for open
				err = errors.New("unexpected closer")
				if e := dec.h(err); e != nil {
					return 0, 0, e
				}
				dec.b.WriteRune(rune) // just copy and ignore

			} else if br.close { // found wrong close bracket
				err = errors.New("mismatched closer")
				if e := dec.h(err); e != nil {
					return 0, 0, e
				}
				dec.b.WriteRune(rune) // just copy and ignore

			} else { // start of nested bracketed string
				dec.b.WriteRune(rune) // copy the open bracket

				// recursively scan to the matching closer
				_, _, err = dec.toBracket(br.other)
				if err != nil {
					return 0, 0, err
				}

				dec.b.WriteRune(br.other) // copy close bracket
			}

		} else { // this rune isn't a bracket
			dec.b.WriteRune(rune) // just copy
		}
	}
}

// Decode one delimited CTS blob from the input stream.
func (dec *Decoder) Decode() (string, rune, string, rune, error) {

	// Read to the first open bracket we fine
	open, close, err := dec.toBracket(0)
	if err != nil {
		return "", 0, "", 0, err
	}
	head := dec.b.String()
	dec.b.Reset()

	// Now read to the matching close bracket,
	// recursively snarfing up nested bracketed substrings along the way.
	_, _, err = dec.toBracket(close)
	if err != nil {
		return "", 0, "", 0, err
	}
	tail := dec.b.String()
	dec.b.Reset()

	return head, open, tail, close, nil
}

// Returns a reader representing the input data remaining
// in the Decoder's buffer.
// The returned reader is valid only until the next call to Decode.
func (dec *Decoder) Buffered() io.Reader {
	return dec.r
}

