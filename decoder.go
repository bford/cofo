package cbe

import (
	"errors"
	"io"
	"bytes"
	"strings"
	"bufio"
	"math/big"
	"encoding/binary"
)

var minusOne = big.NewInt(-1)


// Decoder decodes a series of blobs from an input stream.
type Decoder struct {
	r *bufio.Reader
}

// Create a new Decoder that reads and decodes blobs from r.
// Introduces buffering on r if r is not already a bufio.Reader.
func NewDecoder(r io.Reader) *Decoder {
	d := &Decoder{}
	if br, ok := r.(*bufio.Reader); ok {
		d.r = br
	} else {
		d.r = bufio.NewReader(r)
	}
	return d
}

// Decode the header of the next blob or chunk.
func (d *Decoder) header() (n int, part bool, err error) {
	var h [4]byte

	// first header byte
	h[0], err = d.r.ReadByte()
	if err != nil {
		return 0, false, err
	}
	if h[0] < 0x80 {
		d.r.UnreadByte()	// the header is also the 1-byte content
		return 1, false, nil
	}
	if h[0] != 0x81 {
		return int(h[0] - 0x80), false, nil
	}

	// second header byte
	h[1], err = d.r.ReadByte()
	if err != nil {
		return 0, false, err
	}
	if h[1] >= 0x80 {
		d.r.UnreadByte()	// 1-byte actual content
		return 1, false, nil
	}

	// third header byte
	h[2], err = d.r.ReadByte()
	if err != nil {
		return 0, false, err
	}
	if h[1] < 0x40 {		// 3-byte header for medium-size blob
		return 128 + int(h[1]) << 8 + int(h[2]), false, nil
	}

	// fourth header byte
	h[3], err = d.r.ReadByte()
	if err != nil {
		return 0, false, err
	}
	if h[1] < 0x60 {		// 4-byte header of final large chunk
		return 16384 + int(h[1]&0x1f)<<16 + int(h[2])<<8 + int(h[3]),
			false, nil
	} else {			// 4-byte header of partial blob
		return 16384 + int(h[1]&0x1f)<<16 + int(h[2])<<8 + int(h[3]),
			true, nil
	}
}

// Decode the next complete blob from the input and write it to w.
// Supports streaming: the blob may be arbitrarily long (even infinite).
// This function will decode and write large blobs progressively
// in chunks that may vary in size between MinChunkLen and MaxChunkLen,
// depending on the encoder that wrote the blob.
func (d *Decoder) WriteTo(w io.Writer) (n int64, err error) {
	tot := int64(0)
	for {
		// Decode the next blob or part header
		n, part, err := d.header()
		if err != nil {
			return 0, err
		}

		// Copy the data to the writer
		wn, err := io.CopyN(w, d.r, int64(n))
		if err != nil {
			return 0, err
		}
		if wn != int64(n) {
			return 0, errors.New("short write")
		}
		tot += int64(n)

		if !part {
			return tot, nil
		}
	}
}

// Decode a blob into a byte-slice.
func (d *Decoder) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if _, err := d.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode a blob into a UTF-8 string.
func (d *Decoder) String() (string, error) {
	var sb strings.Builder
	if _, err := d.WriteTo(&sb); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// Decode a blob as a big-endian unsigned integer.
// Returns an error if the decoded value is too large for a uint64.
func (d *Decoder) Uint64() (uint64, error) {
	b, err := d.Bytes()
	if err != nil {
		return 0, err
	}
	if len(b) > 8 {
		return 0, errors.New("integer value too large for uint64")
	}
	var b8 [8]byte
	copy(b8[8-len(b):], b)
	return binary.BigEndian.Uint64(b8[:]), nil
}

// Decode a blob as a big-endian zigzag-encoded signed integer.
// Returns an error if the decoded value is too large for an int64.
func (d *Decoder) Int64() (int64, error) {
	v, err := d.Uint64()
	if err != nil {
		return 0, err
	}

	// zigzag-decode: 0 -> 0, 1 -> -1, 2 -> 1, 3 -> -2, etc.
	if (v & 1) == 0 {
		return int64(v >> 1), nil
	} else {
		return -1 - int64(v >> 1), nil
	}
}

// Decode a blob as a big-endian unsigned integer,
// placing its value into the designated big.Int.
func (d *Decoder) UnsignedInt(v *big.Int) error {
	b, err := d.Bytes()
	if err != nil {
		return err
	}
	v.SetBytes(b)
	return nil
}

// Decode a blob as a big-endian zigzag-encoded signed integer,
// placing its value into the designated big.Int.
func (d *Decoder) SignedInt(v *big.Int) error {
	if err := d.UnsignedInt(v); err != nil {
		return err
	}

	// pull the sign out of the least-significant bit
	sign := v.Bit(0)

	// zigzag-decode: 0 -> 0, 1 -> -1, 2 -> 1, 3 -> -2, etc.
	v.Rsh(v, 1)
	if sign != 0 {
		v.Sub(minusOne, v)
	}

	return nil
}

