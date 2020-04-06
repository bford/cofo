package cts

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type decodeTest struct {
	in              string
	good            bool
	head, tail, rem string
	open, close     rune
}

var decodeTests = []decodeTest{
	{"[]", true, "", "", "", '[', ']'},
	{"foo[bar]", true, "foo", "bar", "", '[', ']'},
	{"foo[bar[blah]]", true, "foo", "bar[blah]", "", '[', ']'},
	{"foo[a[b[c]d[e]f]g]r", true, "foo", "a[b[c]d[e]f]g", "r", '[', ']'},
	{"foo[bar]blah", true, "foo", "bar", "blah", '[', ']'},

	{"", false, "", "", "", 0, 0},
	{"]", false, "", "", "", 0, 0},
	{"foo]", false, "", "", "", 0, 0},
}

// XXX test multi-bracket configs

func TestDecode(t *testing.T) {

	c := Config{} // default config
	buf := bytes.Buffer{}
	for _, dt := range decodeTests {
		d := c.NewDecoder(strings.NewReader(dt.in))
		head, open, tail, close, err := d.Decode()
		if dt.good && err != nil {
			t.Errorf("Decode failed on input '%v'", dt.in)
			continue
		} else if !dt.good && err == nil {
			t.Errorf("Decode should have failed on '%v'", dt.in)
			continue
		}

		// Suck the unread remainder into buf
		if _, err := io.Copy(&buf, d.Buffered()); err != nil {
			t.Errorf("Copy remainder failed: %v", err)
			continue
		}
		rem := buf.String()
		buf.Reset()

		if head != dt.head || open != dt.open ||
			tail != dt.tail || close != dt.close ||
			rem != dt.rem {
			t.Errorf("Decoode produced wrong output %v,%v,%v,%v,%v",
				head, string(open), tail, string(close), rem)
		}

		// XXX check remainder
	}
}
