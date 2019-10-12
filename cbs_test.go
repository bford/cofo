package cbs

import (
	"testing"
	"bytes"
	"math/rand"
)

type testCase struct {
	data, blob []byte
}

var testCases = []testCase{
	{ []byte{},		[]byte{0x80} },		// empty blob

	{ []byte{0},		[]byte{0x00} },		// 1-byte content
	{ []byte{1},		[]byte{0x01} },
	{ []byte{0x7e},		[]byte{0x7e} },
	{ []byte{0x7f},		[]byte{0x7f} },
	{ []byte{0x80},		[]byte{0x81,0x80} },
	{ []byte{0x81},		[]byte{0x81,0x81} },
	{ []byte{0xfe},		[]byte{0x81,0xfe} },
	{ []byte{0xff},		[]byte{0x81,0xff} },

	{ []byte{0x00,0x00},	[]byte{0x82,0x00,0x00} },	// 2-byte
	{ []byte{0xab,0xcd},	[]byte{0x82,0xab,0xcd} },
	{ []byte{0xff,0xff},	[]byte{0x82,0xff,0xff} },

	// 3-byte
	{ []byte{0xab,0xcd,0xef}, []byte{0x83,0xab,0xcd,0xef} },

	// 4-byte
	{ []byte{0xde,0xad,0xbe,0xef}, []byte{0x84,0xde,0xad,0xbe,0xef} }, 

	// 8-byte
	{ []byte{0xde,0xad,0xbe,0xef,0x4b,0xad,0xf0,0x0d},
	  []byte{0x88,0xde,0xad,0xbe,0xef,0x4b,0xad,0xf0,0x0d} }, 

	// 123-byte
	{ []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."),
	  []byte("\xfbLorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.") },

	// 127-byte
	{ []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dol"),
	  []byte("\xffLorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dol") },

	// 128-byte
	{ []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolo"),
	  []byte("\x81\x00\x00Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolo") },

	// 204-byte
	{ []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat."),
	  []byte("\x81\x00\x4cLorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat.") },

	// 778-byte
	{ []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat. Nunc aliquet bibendum enim facilisis gravida. Nisl nunc mi ipsum faucibus vitae aliquet nec ullamcorper. Amet luctus venenatis lectus magna fringilla. Volutpat maecenas volutpat blandit aliquam etiam erat velit scelerisque in. Egestas egestas fringilla phasellus faucibus scelerisque eleifend. Sagittis orci a scelerisque purus semper eget duis. Nulla pharetra diam sit amet nisl suscipit. Sed adipiscing diam donec adipiscing tristique risus nec feugiat in. Fusce ut placerat orci nulla. Pharetra vel turpis nunc eget lorem dolor. Tristique senectus et netus et malesuada."),
	  []byte("\x81\x02\x8aLorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat. Nunc aliquet bibendum enim facilisis gravida. Nisl nunc mi ipsum faucibus vitae aliquet nec ullamcorper. Amet luctus venenatis lectus magna fringilla. Volutpat maecenas volutpat blandit aliquam etiam erat velit scelerisque in. Egestas egestas fringilla phasellus faucibus scelerisque eleifend. Sagittis orci a scelerisque purus semper eget duis. Nulla pharetra diam sit amet nisl suscipit. Sed adipiscing diam donec adipiscing tristique risus nec feugiat in. Fusce ut placerat orci nulla. Pharetra vel turpis nunc eget lorem dolor. Tristique senectus et netus et malesuada.") },

	// 4KiB-byte
	bigTestCase(4096, 0, []byte{0x81,0x0f,0x80}),
	bigTestCase(4096, 1, []byte{0x81,0x0f,0x80}),
	bigTestCase(4096, 0xff, []byte{0x81,0x0f,0x80}),
	bigTestCase(4096, -1, []byte{0x81,0x0f,0x80}),	// random data

	// 16383-byte, largest canonical medium-size blob
	bigTestCase(16383, -1, []byte{0x81,0x3f,0x7f}),
}

var largeCases = []testCase{

	// 16384-byte, smallest non-canonical streamable blob
	bigTestCase(16384, -1, []byte{0x81,0x3f,0x80}),

	// 16511-byte, largest blob representable with 3-byte header
	bigTestCase(16511, -1, []byte{0x81,0x3f,0xff}),

	// 16512-byte, smallest blob requiring 4-byte header
	bigTestCase(16512, -1, []byte{0x81,0x40,0x40,0x00}),
}


func bigTestCase(n, v int, hdr []byte) testCase {
	h := len(hdr)
	buf := make([]byte, h + n)
	copy(buf, hdr)
	if v > 0 {
		for i := range buf[h:] {
			buf[h+i] = byte(v)
		}
	} else if v < 0 {
		rand.Read(buf[h:])
	}
	return testCase{ data: buf[h:], blob: buf }
}


func TestEncode(t *testing.T) {
	var accx, accy []byte
	for i, st := range testCases {

		// Skip tests on large blobs requiring the streaming encoder
		if len(st.data) >= 16384 {
			continue
		}

		// Test encoding into a fresh buffer
		blob := Encode(nil, st.data)
		if bytes.Compare(blob, st.blob) != 0 {
			t.Errorf("incorrect encode in small test %v", i)
		}

		// Test encoding cumulatively
		accx = Encode(accx, st.data)
		accy = append(accy, st.blob...)
		if bytes.Compare(accx, accy) != 0 {
			t.Errorf("incorrect encode in small test %v", i)
		}
	}
}

func TestDecode(t *testing.T) {

	// First just decode each reference blob individually
	for i, st := range testCases {
		data, rem, err := Decode(st.blob)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(data, st.data) != 0 {
			t.Errorf("incorrect decode in small test %v", i)
		}
		if len(rem) != 0 {
			t.Errorf("failed to consume everything in test %v", i)
		}
	}

	// concatenate all the small-test reference blobs
	var buf []byte
	for _, st := range testCases {
		buf = append(buf, st.blob...)
	}

	// now progressively decode and check each
	for i, st := range testCases {
		data, rem, err := Decode(buf)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(data, st.data) != 0 {
			t.Errorf("incorrect decode in small test %v", i)
		}
		buf = rem
	}
	if len(buf) != 0 {
		t.Error("failed to decode everything in concatenated decode")
	}
}

func TestEncoder(t *testing.T) {
	var buf bytes.Buffer
	var ref []byte
	enc := NewEncoder(&buf)

	// Encode all our test cases consecutively with one Encoder
	for i, st := range testCases {
		if err := enc.Bytes(st.data); err != nil {
			t.Error(err)
		}
		ref = append(ref, st.blob...)

		if bytes.Compare(buf.Bytes(), ref) != 0 {
			t.Errorf("incorrect encode in case %v len %v",
				 i, len(st.data))
		}
	}
	if bytes.Compare(buf.Bytes(), ref) != 0 {
		t.Errorf("incorrect encode")
	}
}

