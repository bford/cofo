package cbs

import (
	"errors"
	"io"
	"bytes"
	"strings"
	"encoding/binary"
	"math/big"
)


// Minimum streamable chunk size allowed by the encoding.
const MinChunkLen int = 0x4000		// 16384 bytes

// Maximum streamable chunk size allowed by the encoding.
const MaxChunkLen int = 0x203fff	// 2,113,535 bytes

const defaultChunkLen = MinChunkLen



// An Encoder encodes a series of blobs to an output stream.
type Encoder struct {
	w io.Writer
	buf []byte
}

// Create a new Encoder that writes encoded blobs to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode a blob by reading bytes from r until encountering EOF.
// Supports streaming:
// r can represent arbitrarily many bytes (even infinite).
// This function will 
func (e *Encoder) ReadFrom(r io.Reader) (n int64, err error) {

	// Get our chunk buffer, creating it if needed
	buf := e.getBuf()
	chunkLen := len(buf) - 4

	tot := int64(0)
	more := true
	for more {
		// Read a full chunk into the chunk buffer or until EOF
		l, err := io.ReadFull(r, buf[4:])
		if err != nil && err != EOF && err != io.ErrUnexpectedEOF {
			println("ReadFull l", l, "err", err)
			return 0, err
		}

		// Write either a complete or partial blob
		more = false
		h := 0
		if l == chunkLen && err == nil { // chunk-size partial blob
			n := l - 16384
			buf[0] = 0x81
			buf[1] = 0x60 + byte(n >> 16)
			buf[2] = byte(n >> 8)
			buf[3] = byte(n)
			more = true

		} else if l == 1 && buf[4] < 128 { // 1-byte no-header blob
			h = 4

		} else if l < 128 { // short blob with 1-byte header
			buf[3] = byte(0x80 + l)
			h = 3

		} else if l < 16384+128 { // medium blob with 3-byte header
			n := l - 128
			buf[1] = byte(0x81)
			buf[2] = byte(n >> 8)
			buf[3] = byte(n)
			h = 1

		} else {		// large blob with 4-byte header
			n := l - 16384
			buf[0] = 0x81
			buf[1] = 0x40 + byte(n >> 16)
			buf[2] = byte(n >> 8)
			buf[3] = byte(n)
		}

		// Write the blob header and data from the buffer
		lw, err := e.w.Write(buf[h:4+l])
		if err != nil {
			return 0, err
		} else if lw != 4-h+l {
			return 0, errors.New("short write")
		}

		tot += int64(l)
	}
	return tot, nil
}

// Encode a byte-slice as a blob.
func (e *Encoder) Bytes(b []byte) error {
	_, err := e.ReadFrom(bytes.NewReader(b))
	return err
}

// Encode a UTF-8 string as a blob.
func (e *Encoder) String(s string) error {
	_, err := e.ReadFrom(strings.NewReader(s))
	return err
}

// Encode a uint64 as a big-endian unsigned integer blob.
func (e *Encoder) Uint64(v uint64) error {
	var b8 [8]byte
	b := b8[:]
	binary.BigEndian.PutUint64(b, v)
	for len(b) > 0 && b[0] == 0 {	// trim leading 0 bytes
		b = b[1:]
	}
	return e.Bytes(b)
}

// Encode an int64 as a big-endian zigzag-encoded signed integer blob.
func (e *Encoder) Int64(v int64) error {
	var u uint64
	if v >= 0 {
		u = uint64(v) << 1
	} else {
		u = uint64(^v) << 1 + 1
	}
	return e.Uint64(u)
}

// Encode the absolute value of a big.Int
// as a big-endian unsigned integer blob.
func (e *Encoder) UnsignedInt(v *big.Int) error {
	return e.Bytes(v.Bytes())
}

// Encode a big.Int as a big-endian zigzag-encoded signed integer blob.
func (e *Encoder) SignedInt(v *big.Int) error {
	u := &big.Int{}
	if v.Sign() >= 0 {
		u.Lsh(v, 1)
	} else {
		u.Sub(minusOne, v)
		u.Lsh(u, 1)
		u.SetBit(u, 0, 1)
	}
	return e.Bytes(u.Bytes())
}


// Get the current chunk buffer, creating one if necessary.
func (e *Encoder) getBuf() []byte {
	if e.buf == nil {
		e.buf = make([]byte, 4 + defaultChunkLen)
	}
	return e.buf
}

// Returns the current chunk size used in streaming operation.
func (e *Encoder) ChunkLen() int {
	buf := e.getBuf()
	return len(buf) - 4
}

// Set the chunk size used for streaming operation.
// Larger chunks incur slightly less encoding overhead
// but require larger buffers.
// Panics if chunkLen is not between MinChunkLen and MaxChunkLen.
func (e *Encoder) SetChunkLen(chunkLen int) {
	if chunkLen < MinChunkLen {
		panic("chunk size too small")
	}
	if chunkLen > MaxChunkLen {
		panic("chunk size too large")
	}

	bufLen := 4 + chunkLen	// for header plus chunk
	if cap(e.buf) < bufLen {
		e.buf = make([]byte, bufLen)
	}
	e.buf = e.buf[:bufLen]
}

