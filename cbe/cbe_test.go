package cbe

import (
	"bytes"
	"math/rand"
	"testing"
)

type testCase struct {
	data, blob []byte
}

var testCases = []testCase{
	{[]byte{}, []byte{0x80}}, // empty blob

	{[]byte{0}, []byte{0x00}}, // 1-byte content (#1)
	{[]byte{1}, []byte{0x01}},
	{[]byte{0x7e}, []byte{0x7e}},
	{[]byte{0x7f}, []byte{0x7f}},
	{[]byte{0x80}, []byte{0x81, 0x80}},
	{[]byte{0x81}, []byte{0x81, 0x81}},
	{[]byte{0xfe}, []byte{0x81, 0xfe}},
	{[]byte{0xff}, []byte{0x81, 0xff}},

	{[]byte{0x00, 0x00}, []byte{0x82, 0x00, 0x00}}, // 2-byte (#9)
	{[]byte{0xab, 0xcd}, []byte{0x82, 0xab, 0xcd}},
	{[]byte{0xff, 0xff}, []byte{0x82, 0xff, 0xff}},

	// 3-byte (#12)
	{[]byte{0xab, 0xcd, 0xef}, []byte{0x83, 0xab, 0xcd, 0xef}},

	// 4-byte
	{[]byte{0xde, 0xad, 0xbe, 0xef}, []byte{0x84, 0xde, 0xad, 0xbe, 0xef}},

	// 8-byte
	{[]byte{0xde, 0xad, 0xbe, 0xef, 0x4b, 0xad, 0xf0, 0x0d},
		[]byte{0x88, 0xde, 0xad, 0xbe, 0xef, 0x4b, 0xad, 0xf0, 0x0d}},

	// 63-byte boundary case (#15)
	{[]byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do"),
		[]byte("\xbfLorem ipsum dolor sit amet, consectetur adipiscing elit, sed do")},

	// 64-byte boundary case (#16)
	{[]byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do "),
		[]byte("\xc0\x00Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do ")},

	// 204-byte (#17)
	{[]byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat."),
		[]byte("\xc0\x8cLorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat.")},

	// 778-byte (#18)
	{[]byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat. Nunc aliquet bibendum enim facilisis gravida. Nisl nunc mi ipsum faucibus vitae aliquet nec ullamcorper. Amet luctus venenatis lectus magna fringilla. Volutpat maecenas volutpat blandit aliquam etiam erat velit scelerisque in. Egestas egestas fringilla phasellus faucibus scelerisque eleifend. Sagittis orci a scelerisque purus semper eget duis. Nulla pharetra diam sit amet nisl suscipit. Sed adipiscing diam donec adipiscing tristique risus nec feugiat in. Fusce ut placerat orci nulla. Pharetra vel turpis nunc eget lorem dolor. Tristique senectus et netus et malesuada."),
		[]byte("\xc2\xcaLorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Dolor sed viverra ipsum nunc aliquet bibendum enim. In massa tempor nec feugiat. Nunc aliquet bibendum enim facilisis gravida. Nisl nunc mi ipsum faucibus vitae aliquet nec ullamcorper. Amet luctus venenatis lectus magna fringilla. Volutpat maecenas volutpat blandit aliquam etiam erat velit scelerisque in. Egestas egestas fringilla phasellus faucibus scelerisque eleifend. Sagittis orci a scelerisque purus semper eget duis. Nulla pharetra diam sit amet nisl suscipit. Sed adipiscing diam donec adipiscing tristique risus nec feugiat in. Fusce ut placerat orci nulla. Pharetra vel turpis nunc eget lorem dolor. Tristique senectus et netus et malesuada.")},

	// 4KiB-byte
	bigTestCase(4096, 0, []byte{0xcf, 0xc0}), // (#19)
	bigTestCase(4096, 1, []byte{0xcf, 0xc0}),
	bigTestCase(4096, 0xff, []byte{0xcf, 0xc0}),
	bigTestCase(4096, -1, []byte{0xcf, 0xc0}), // random data

	// 16384-byte (#23)
	bigTestCase(16384, -1, []byte{0xff, 0xc0}),

	// 16447-byte, largest canonical medium-size blob (#24)
	bigTestCase(16447, -1, []byte{0xff, 0xff}),

	// 16448-byte, smallest large blob with 4-byte header (#25)
	bigTestCase(16448, -1, []byte{0x81, 0x00, 0x00, 0x00}),

	// 32768-byte (#26)
	bigTestCase(32768, -1, []byte{0x81, 0x00, 0x3f, 0xc0}),

	// 4210750-byte (#27)
	bigTestCase(4210750, -1, []byte{0x81, 0x3f, 0xff, 0xfe}),

	// 4210751-byte (#27)
	//bigTestCase(4210751, -1, []byte{0x81,0x3f,0xff, 0xff}),
	// XXX Encoder breaks it into 2 chunks; need to account for that...
}

// XXX tests could be better: e.g., need error cases too...

func bigTestCase(n, v int, hdr []byte) testCase {
	h := len(hdr)
	buf := make([]byte, h+n)
	copy(buf, hdr)
	if v > 0 {
		for i := range buf[h:] {
			buf[h+i] = byte(v)
		}
	} else if v < 0 {
		rand.Read(buf[h:])
	}
	return testCase{data: buf[h:], blob: buf}
}

func TestEncode(t *testing.T) {
	var accx, accy []byte
	for i, st := range testCases {

		// Skip tests on large blobs requiring the streaming encoder
		if len(st.data) >= 16448 {
			continue
		}

		// Test encoding into a fresh buffer
		blob := Encode(nil, st.data)
		if bytes.Compare(blob, st.blob) != 0 {
			t.Errorf("incorrect encode in test %v", i)
		}

		// Test encoding cumulatively
		accx = Encode(accx, st.data)
		accy = append(accy, st.blob...)
		if bytes.Compare(accx, accy) != 0 {
			t.Errorf("incorrect encode in test %v", i)
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
			t.Errorf("incorrect decode in test %v", i)
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
			t.Errorf("incorrect decode in test %v", i)
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
	enc.SetChunkLen(MaxChunkLen)

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

func TestDecoder(t *testing.T) {

	// Decode each test case individually
	var acc []byte
	for i, st := range testCases {
		dec := NewDecoder(bytes.NewReader(st.blob))
		b, err := dec.Bytes()
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(b, st.data) != 0 {
			t.Errorf("incorrect decode in case %v len %v",
				i, len(st.data))
		}

		acc = append(acc, st.blob...)
	}

	// Decode all the test cases consecutively from one buffer
	dec := NewDecoder(bytes.NewReader(acc))
	for i, st := range testCases {
		b, err := dec.Bytes()
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(b, st.data) != 0 {
			t.Errorf("incorrect decode in case %v len %v",
				i, len(st.data))
		}
	}
}
