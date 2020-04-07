package cri

import (
	"testing"
)

var lazyIRI = &Form{Unicode: true, Lazy: true}
var lazyCRI = &Form{Unicode: true, Brackets: true, NestedIP: true, Lazy: true}

type testFromCase struct {
	src, dst string
	form     *Form
}

var testFromCases = []testFromCase{

	// Some basic no-conversion cases
	{"https://foo.bar/", "https://foo.bar/", URI},
	{"https://foo.bar/", "https://foo.bar/", lazyCRI},
	{"https[//foo.bar/]", "https[//foo.bar/]", CRI},

	// Basic cross-conversions (#3)
	{"https[//foo.bar/]", "https://foo.bar/", URI},
	{"https[//foo.bar/]", "https[//foo.bar/]", lazyCRI},
	{"https://foo.bar/", "https[//foo.bar/]", CRI},

	// IPv4 address conversions and non-conversions (#6)
	{"https://12.34.56.78/", "https://12.34.56.78/", URI},
	{"https://12.34.56.78/", "https://12.34.56.78/", lazyCRI},
	{"https://12.34.56.78/", "https[//ip4[12.34.56.78]/]", CRI},
	{"https://ip4[12.34.56.78]/", "https://12.34.56.78/", URI},
	{"https://ip4[12.34.56.78]/", "https://ip4[12.34.56.78]/", lazyCRI},
	{"https://ip4[12.34.56.78]/", "https[//ip4[12.34.56.78]/]", CRI},

	// IPv6 address conversions and non-conversions (#12)
	{"https://[a:b::c:d]/", "https://[a:b::c:d]/", URI},
	{"https://[a:b::c:d]/", "https://[a:b::c:d]/", lazyCRI},
	{"https://[a:b::c:d]/", "https[//ip6[a:b::c:d]/]", CRI},
	{"https://ip6[a:b::c:d]/", "https://[a:b::c:d]/", URI},
	{"https://ip6[a:b::c:d]/", "https://ip6[a:b::c:d]/", lazyCRI},
	{"https://ip6[a:b::c:d]/", "https[//ip6[a:b::c:d]/]", CRI},

	// XXX need a lot more
}

// Test conversions
func TestFrom(t *testing.T) {
	for i, c := range testFromCases {
		out, err := c.form.From(c.src)
		switch {
		case c.dst != "" && err != nil:
			t.Error("case", i, "error", err)
		case c.dst != "" && out != c.dst:
			t.Error("case", i, "expecting", c.dst, "but got", out)
		case c.dst == "" && err == nil:
			t.Error("case", i, "expecting error but got", out)
		}
	}
}
