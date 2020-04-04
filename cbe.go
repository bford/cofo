// Package cbe-go implements Composable Binary Encoding (CBE),
// which efficiently embeds one arbitrary-length binary string in another
// so that a decoder can efficiently find the embedded string's length.
//
// For a detailed introduction to CBE,
// see the draft blog post at https://bford.info/draft/cbe/
// (warning: this is a temporary link that will change).
//
// The plain functions Encode and Decode operate on
// contiguous in-memory byte slices,
// and do not support streaming.
// The Encode and Decode types provide stream-oriented encoding and decoding,
// supporting arbitrary-length byte strings including infinite streams.
//
package cbe

import (
	"errors"
	"io"
)

// Encode a byte slice src and append it to slice dst.
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
		return append(dst, src...) // value is the blob encoding
	}
	if n < 64 { // shorter than 2^7 bytes
		dst := append(dst, byte(128+n))
		return append(dst, src...)
	}

	// 2-byte header encoding
	if n < 16448 { // shorter than 2^6+2^14 bytes
		n -= 64
		dst := append(dst, 0xc0+byte(n>>8), byte(n))
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
	if buf[0] < 128 { // 0vvvvvvv direct value encoding
		return 0, 1, nil // the header is the 1-byte content
	}
	if buf[0] != 129 && buf[0] < 128+64 { // 10nnnnnn encoding for n != 1
		return 1, int(buf[0] - 128), nil
	}

	// 2-byte headers
	if len(buf) < 2 {
		return 0, 0, EOF
	}
	if buf[0] == 129 && buf[1] >= 128 { // 10000001,1vvvvvvv encoding
		return 1, 1, nil
	}
	if buf[0] >= 128+64 { // 11nnnnnn,nnnnnnnn encoding
		return 2, 64 + (int(buf[0]&63) << 8) + int(buf[1]), nil
	}

	// 4-byte streaming-capable headers
	return 0, 0, TooLong
}

// Decode a small blob, whose content is of size less than 16448 bytes,
// from the start of a byte slice.
// On success, returns a sub-slice of the blob containing the content,
// and a disjoint sub-slice containing the remainder of the buffer
// immediately following the decoded blob.
//
// This function returns EOF if the provided byte string
// does not contain a complete blob.
//
// This decoding function does not support large blobs of 16448 bytes or more,
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
	return buf[ofs : ofs+n], buf[ofs+n:], nil
}

var EOF = io.EOF

var TooLong = errors.New("blob content too long")
