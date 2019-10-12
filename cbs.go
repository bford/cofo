package cbs

import (
	"errors"
	"io"
)

// Blob-encode a byte slice src and append it to slice dst.
// Allocates and returns a new destination buffer
// if the blob-encoded data does not fit into dst.
//
// This function supports encoding only small slices
// whose content is less than 16KiB in size.
// To encode large slices, use the streaming-capable Encoder.
//
func Encode(dst, src []byte) []byte {

	// 1-byte header encoding
	n := len(src)
	if n == 1 && src[0] < 128 {
		return append(dst, src...)	// value is the blob encoding
	}
	if n < 128 {				// shorter than 2^7 bytes
		dst := append(dst, byte(128+n))
		return append(dst, src...)
	}

	// 3-byte header encoding
	if n < 16384 {				// shorter than 2^7+2^16 bytes
		n -= 128
		dst := append(dst, 0x81, byte(n >> 8), byte(n))
		return append(dst, src...)
	}

	panic("content too large for small-blob encoder")
}

// Decode the header of a small blob, whose content is of size less than 16KiB,
// from the start of a byte slice.
// On success, returns the offset in the byte slice
// and the length in bytes of the blob's content.
//
// This function returns EOF if the provided byte string
// does not contain a complete blob header.
//
// This decoding function does not support large blobs of 16KiB or more,
// which may require the decoder to handle multi-part streaming encodings.
// On see a blob header for a large blob, this function returns an error.
// To decode large blobs of 16KiB or more, use the streaming-capable Decoder.
// 
func decodeHeader(buf []byte) (contentOfs, contentLen int, err error) {

	// 1-byte headers
	if len(buf) == 0 {
		return 0, 0, EOF
	}
	if buf[0] < 128 {		// 0vvvvvvv direct value encoding
		return 0, 1, nil	// the header is the 1-byte content
	}
	if buf[0] != 129 {		// 1nnnnnnn encoding for n != 1
		return 1, int(buf[0]-128), nil
	}

	// 2-byte headers
	if len(buf) < 2 {
		return 0, 0, EOF
	}
	if buf[1] >= 128 {		// 10000001,1vvvvvvv encoding
		return 1, 1, nil
	}

	// 3-byte headers
	if len(buf) < 3 {
		return 0, 0, EOF
	}
	if buf[1] < 64 {		// 10000001,00nnnnnn,nnnnnnnn
		return 3, 128 + (int(buf[1]) << 8) + int(buf[2]), nil
	}

	// 4-byte streaming-capable headers
	return 0, 0, TooLong
}

// Decode the header of a small blob, whose content is of size less than 16KiB,
// from the start of a byte slice.
// On success, returns a sub-slice of the blob containing the content,
// and a disjoint sub-slice containing the remainder of the buffer
// immediately following the decoded blob.
//
// This function returns EOF if the provided byte string
// does not contain a complete blob.
//
// This decoding function does not support large blobs of 16KiB or more,
// which may require the decoder to handle multi-part streaming encodings.
// On see a blob header for a large blob, this function returns an error.
// To decode large blobs of 16KiB or more, use the streaming-capable Decoder.
// 
func Decode(buf []byte) (content, remainder []byte, err error) {
	ofs, n, err := decodeHeader(buf)
	if err != nil {
		return nil, nil, err
	}
	if len(buf) < ofs+n {
		return nil, nil, EOF
	}
	return buf[ofs:ofs+n], buf[ofs+n:], nil
}

var EOF = io.EOF

var TooLong = errors.New("blob content too long")

