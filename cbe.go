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
	"bytes"
	"io"
)

// Encode a byte slice src and append its CBE encoding to slice dst.
// Allocates and returns a new destination buffer
// if the blob-encoded data does not fit into dst.
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

	// 4-byte header encoding
	if n < 4210752 { // shorter than 2^6+2^14+2^22 bytes
		n -= 16448
		dst := append(dst, 0x81, 0x00+byte(n>>16),
			byte(n>>8), byte(n))
		return append(dst, src...)
	}

	// For really large encodes, fall back on the general Encoder.
	// Use maximum-size chunks since everything's in-memory anyway.
	buf := bytes.NewBuffer(dst)
	enc := NewEncoder(buf)
	enc.SetChunkLen(MaxChunkLen)
	err := enc.Bytes(src)
	if err != nil {
		panic("encoding error: " + err.Error())
	}
	return buf.Bytes()
}

// Decode a blob header from the start of a byte slice.
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
func decodeHeader(buf []byte) (dataOfs, dataLen int, part bool, err error) {

	// 1-byte headers
	if len(buf) == 0 {
		return 0, 0, false, EOF
	}
	if buf[0] < 128 { // 0vvvvvvv direct value encoding
		return 0, 1, false, nil // the header is the 1-byte content
	}
	if buf[0] != 129 && buf[0] < 128+64 { // 10nnnnnn encoding for n != 1
		return 1, int(buf[0] - 128), false, nil
	}

	// 2-byte headers
	if len(buf) < 2 {
		return 0, 0, false, EOF
	}
	if buf[0] == 129 && buf[1] >= 128 { // 10000001,1vvvvvvv encoding
		return 1, 1, false, nil
	}
	if buf[0] >= 128+64 { // 11nnnnnn,nnnnnnnn encoding
		return 2, 64 + int(buf[0]&63)<<8 + int(buf[1]), false, nil
	}

	// 4-byte headers
	if len(buf) < 4 {
		return 0, 0, false, EOF
	}
	part = buf[1] >= 0x40
	return 4, 16448 + int(buf[1]&0x3f)<<16 +
		int(buf[2])<<8 + int(buf[3]), part, nil
}

// Decode a blob from the start of a byte slice,
// returning a byte slice containing the blob's encoded content,
// and a disjoint slice containing the remainder of the buffer
// immediately following the decoded blob.
//
// On successfully decoding a blob that was encoded in only one chunk,
// including all small blobs less than 16448 bytes,
// returns a sub-slice of the original buffer containing the blob's content.
// The blob's actual content is never actually copied in this case.
//
// When decoding a large blob that was encoded into multiple chunks,
// copying is necessary to concatenate the chunk payloads into one,
// so Decode returns a fresh byte slice containing this concatenated content.
//
// This function returns EOF if the provided byte string
// does not contain a complete blob.
//
func Decode(buf []byte) (content, remainder []byte, err error) {
	for {
		ofs, n, part, err := decodeHeader(buf)
		if err != nil {
			return nil, nil, err
		}
		if len(buf) < ofs+n {
			return nil, nil, EOF
		}

		if !part && content == nil { // all in only one chunk
			return buf[ofs : ofs+n], buf[ofs+n:], nil
		}

		// Concatenate all content into one buffer
		content = append(content, buf[ofs:ofs+n]...)
		buf = buf[ofs+n:]
		if !part { // final chunk
			return content, buf, nil
		}
	}
}

var EOF = io.EOF

