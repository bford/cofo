# Composable Binary Encoding (CBE)

We often want to embed a veriable-length binary string
into a longer string or stream,
so that a decoder can find the end of the embedded string unambiguously.
*Composable binary encoding* or *CBE* is an encoding scheme
designed to solve this (and only this) problem simply and efficiently.
Given an arbitrary byte string of any length from zero to infinity (endless),
CBE embeds it in a larger byte sequence such that:

*	Embedding is space-efficient for small embedded strings,
	incurring:
	*	0-byte overhead when the embedded string is
		a 7-bit integer or single ASCII character.
	*	1-byte overhead for embedded strings less than 64 bytes.
	*	2-byte overhead for embedded strings less than 16448 bytes.
*	Streaming encoders can delimit arbitrarily-long embedded strings
	progressively in variable-size chunks between &sim;16KB and &sim;4MB.
*	Relative space overhead diminishes rapidly for large strings,
	to a limit of &sim;0.0001% when streaming with the maximum chunk size.
*	Every byte of the embedded string appears verbatim and in-order
	in the CBE-encoding, with no transformation or obfuscation
	to interfere with manual inspection or search tools.
*	There is only one valid CBE encoding
	for embedded strings of 16KiB or less,
	making CBE embedding automatically
	[*canonical*](https://en.wikipedia.org/wiki/Canonicalization)
	in this common case.
*	Both encoding and decoding is extremely simple,
	with only a few cases,
	and even simpler when embedded strings are constrained to be short.

A [prototype implementation in Go](https://github.com/bford/cbe-go)
is already available,
and porting CBE to other language should be straightforward.


## Encoding binary blobs into one or more chunks

CBE takes an arbitrary byte sequence or *blob* of any length &ndash;
even an endless stream &ndash;
and encodes it so that when embedded in another "container" byte stream,
a decoder can unambiguously find the blob's end (if it has one).
CBE has no notion of data types aside from byte sequences
and does not know or care what you put in an embedded blob:
it assumes the context in which the CBE-encoded blob appears
will determine that.
CBE's goal is to do only one thing well,
namely embed variable-length byte sequences efficiently.

While generalizing to arbitrary-length embedded blobs,
CBE optimizes for the important common case of *short* strings
in terms of both space-efficiency and encoding/decoding simplicity.
To this end, CBE categorizes blobs into two general size categories,
*small* and *large*.
Small blobs encode strings of up to 16447 bytes
in a single contiguous chunk.
Large blobs encode strings of 16448 bytes or more
into one or more successive chunks to support streaming operation.
This way, an encoder can produce *partial* chunks of large blobs progressively
without knowing how many more chunks there will be
until it produces the one *final* chunk.

### CBE chunk headers

Each chunk comprising a CBE blob
contains a *header* of 1&ndash;4 bytes
and a *payload* of up to 4,210,751 bytes
(2<sup>6</sup>+2<sup>14</sup>+2<sup>22</sup>-1).
The payload immediately follows the header,
except in the case of 1-byte payloads,
which are part of the header itself.
There are seven header encodings total, as follows:

<table align="center">
<tr align="left"><th> Header encoding <th> Description		<th> Payload
<tr><td> <tt>0b10000000</tt>	<td> 0-byte empty final chunk	<td> none
<tr><td> <tt>0b0<i>vvvvvvv</i></tt>
		<td> 1-byte final chunk with payload <tt>0b0<i>vvvvvvv</i></tt> (0-127)
				<td> 1 byte within header
<tr><td> <tt>0b10000001,1<i>vvvvvvv</i></tt>
		<td> 1-byte final chunk with payload <tt>0b1<i>vvvvvvv</i></tt> (128-255) &nbsp;
				<td> 1 byte within header
<tr><td> <tt>0b10<i>nnnnnn</i></tt>
		<td> final chunk of payload length <i>n</i>
			with 6-bit <i>n</i>
				<td> 2&ndash;63 bytes following header
<tr><td> <tt>0b11<i>nnnnnn</i>,<i>nnnnnnnn</i></tt>
		<td> final chunk of payload length 64+<i>n</i>
			with 14-bit <i>n</i>
				<td> 64&ndash;16447 bytes following header
<tr><td> <tt>0b10000001,00<i>nnnnnn</i>,<i>nnnnnnnn</i>,<i>nnnnnnnn</i></tt>
		<td> final chunk of payload length 16448+<i>n</i>
			with 22-bit <i>n</i>
				<td> 16448&ndash;4,210,751 bytes
					following header
<tr><td> <tt>0b10000001,01<i>nnnnnn</i>,<i>nnnnnnnn</i>,<i>nnnnnnnn</i></tt> &nbsp;
		<td> partial chunk of payload length 16448+<i>n</i>
			with 22-bit <i>n</i>
				<td> 16448&ndash;4,210,751 bytes
					following header
</table>

In the last three header encodings,
the bits comprising the value *n* are in big-endian byte order,
with the most-significant bits in the second header byte.

The first header variant is really just a special case of the fourth,
in which the 6-bit length *n* is zero,
and accordingly, zero payload bytes follow the 1-byte header.
Encoders and decoders therefore need not actually distinguish these two cases.
The above table shows the empty-blob case separately
only for clarity of presentation.

### Small blobs

CBE optimizes small blobs for space-efficiency
by incurring at most one byte of header overhead
when the payload is less than 64 bytes long,
and only two bytes of overhead for all small blobs
containing less than 16448 bytes.
In the special case of a blob containing a 1-byte payload
whose most-significant bit is zero &ndash;
such as a small integer in the range 0&ndash;127
or a single ASCII character &ndash;
CBE encodes the header *and* payload into a single byte
using the second header variant above.
In this case, the payload *is* the header.

In every case including 1-byte payloads embedded within the header,
all payload bytes appear verbatim and contiguously
either within or immediately following the header,
with no escaping or other transformation of payload bytes.
This property avoids unnecessary obfuscation of embedded byte sequences,
which is useful when manually inspecting a
[hex dump](https://en.wikipedia.org/wiki/Hex_dump)
or searching binary data
for text [strings](https://en.wikipedia.org/wiki/Strings_(Unix))
or other embedded content for example.
Avoiding payload transformation
also ensures that CBE decoders often need not copy payloads,
but can leave them in the input buffer
and simply pass a pointer and length &ndash;
or a slice in modern systems languages like
[Go](https://golang.org) and
[Rust](https://www.rust-lang.org) &ndash;
to some function that consumes the chunk's payload.

Small blobs are automatically
[*canonical*](https://en.wikipedia.org/wiki/Canonicalization)
or 
[*distinguished*](https://en.wikipedia.org/wiki/X.690#DER_encoding),
in that there is only one possible way to encode any blob
containing up to 16447 payload bytes.
This is because the header encodings with more length bits
offset the length value *n* to ensure that longer headers
cannot redundantly indicate the same payload length as a shorter header
for small values of *n*.
Canonical encoding is useful in applications such as cryptography,
where it is often essential that all encoders of particular data
all arrive at one and only unique binary encoding of that data,
for [digital signing](https://en.wikipedia.org/wiki/Digital_signature)
and verification for example.
In this sense,
CBE can serve the same purpose as the
[X.690 distinguished encoding rules (DER)](https://en.wikipedia.org/wiki/X.690#DER_encoding),
but with a simpler and more efficient encoding.

Because the only encoding for *partial* chunks
uses the final 4-byte header variant above
with a minimum payload size of 16448 bytes,
small blobs less than this size are always contiguous
and can never be split into multiple chunks.
This property ensures that in formats and protocols
that CBE-encode data items known &ndash; or deliberately constrained &ndash;
to be less than 16448 bytes,
the decoder need not incur the code complexity
of even knowing how to decode multi-chunk blobs.
In this way, CBE optimizes not just space-efficiency
but also implementation simplicity
in the common case of small blobs.


### Large blobs

Blobs embedding strings 16448 bytes or larger
may be split into zero or more *partial* chunks
followed by exactly one *final* chunk.
Blobs containing 4,210,752 bytes or more *must* be split in this way.
All partial chunks use the last header encoding above,
so that each non-final chunk contains a payload of between
16448 and 4,210,751 bytes inclusive.
The final chunk comprising a blob
can use any of the final chunk encodings defined above,
and thus can contain payloads between 0 and 4,210,751 bytes.

Streaming a long blob in minimum-size chunks
incurs four bytes of metadata overhead every 16448 bytes of content,
for an overhead of &sim;0.2%.
Using maximum-size chunks, in contrast,
yields an overhead of less than 0.0001%.

This chunk sizing flexibility
allows streaming encoders to choose a balance
between the space and processing efficiency of using large chunks,
versus the internal memory requirement of buffering a complete chunk
and the latency that this buffering may add to streaming applications.
The supported range of chunk sizes is inevitably somewhat arbitrary,
but chosen to correspond roughly to the range of chunk and block sizes
most frequently used in storing or processing bulk data
in many typical formats and protocols:
e.g., the 16-byte records typical of [TLS](https://en.wikipedia.org/wiki/Transport_Layer_Security),
the 4KiB&ndash;1MiB block sizes commonly used in clustered storage systems
or [IPFS](https://en.wikipedia.org/wiki/InterPlanetary_File_System),
the 64&ndash;256KiB block sizes typical of 
[flash memories](https://en.wikipedia.org/wiki/Flash_memory),
the 32KiB window used in [gzip](https://en.wikipedia.org/wiki/Gzip) compression
to the 100-900KiB block sizes of [bzip2](https://en.wikipedia.org/wiki/Bzip2),
etc.

One cost of this chunk size flexibility in large blobs
is that decoders must be prepared to decode and combine chunks of varying size.
A particular context or data format using CBE
can impose restrictions on the range 
If CBE's range of chunk sizes is too broad
for a particular format or protocol using CBE,
that protocol can further constrain the range of allowed chunk sizes
in that particular context.
A protocol using CBE could restrict chunks to be strict powers of two
between 2<sup>15</sup> (32KiB) and 2<sup>22</sup> (4MiB), for example,
or a more restrictive range.

Unlike small blobs,
the encodings of large blobs are no longer automatically canonical,
since different encoders may split blobs with different chunk boundaries.
This does not mean large blobs cannot be *made* canonical, of course.
A particular data format using CBE can require a fixed chunk size
in a context requiring a canonical encoding,
thereby achieving the properties of 
[DER](https://en.wikipedia.org/wiki/X.690#DER_encoding)
for large blobs as well as small.

The design choice to make small blobs automatically canonical,
while allowing chunking flexibility for large blobs,
reflects a balancing of priorities:
toward the simplicity and space-efficiency of 
a simple, contiguious, and unique representation for small data items
such as common integers, strings, and other metadata
usually fitting in small blobs,
and for which streaming is usually unnecessary;
and in contrast
towards support for varying stream processing efficiency tradeoffs
for the bulk data exchange and processing
that large blobs are intended to support.


## Suggested blob encodings for common data types

CBE does not know or care what you put in a blob;
that is the point of the type-agnostic byte-string approach.
However, CBE was designed in the expectation
that blobs would frequently hold data of various extremely common data types,
such as integers, strings, or key/value pairs.
This section discusses some practices and suggestions
for such common-case uses of blobs,
without in any way intending to restrictive or prescriptive.


### Integer blobs

Integers are one of the most common basic data types
used throughout innumerable data formats and protocols.
It is simple and natural to use CBE blobs
to encode variable-length integers efficiently,
as an alternative to base-7
[varints](https://developers.google.com/protocol-buffers/docs/encoding)
for example.
This section discusses this use first for unsigned, then signed integers.


#### Unsigned integers

Blob-encoding a variable-length unsigned integer is easy in principle:
simply serialize the integer 8 bits at a time into a byte sequence,
then CBE-encode that byte sequence.

CBE does not care whether you serialize the integer
in big-endian or little-endian byte order.
I recommend big-endian, however,
for consistency with the "network byte order" tradition,
and so that encoded integers are maximally recognizable and legible
in manual inspection for example.
Because CBE avoids transforming the payload of a blob, for example,
encoding the 64-bit integer 0xDEADBEEF4BADF00D this way
will be fairly recognizable in a hex dump:

		... 88 DE AD BE EF 4B AD F0 0D ...

When we blob-encode a serialized integer,
we automatically get an extremely compact, 1-byte encoding
of small integer values less than 128,
an important common case in many situations.
For example, data formats and protocols often encode
data and message type codes, keys, enumeration values,
and the like as small integer constants.

Blob encoding incurs further overhead extremely gradually,
representing integers of up to 504 bits with only 1-byte overhead
for example,
and representing larger integers commonly used in cryptography
(e.g., 4096-bit RSA numbers) with only 2-byte overhead.


#### Signed integers

CBE again does not particularly care how signed integers are serialized.
However, the
[ZigZag encoding](...)
used in
Protobuf and other formats is also well-matched to CBE.
This encoding transforms a signed integer into an unsigned integer,
which we then serialize and CBE-encode as discussed above.

In contrast with standard two's-complement integers,
ZigZag encoding makes uses the integer's *least*-significant bit
as the sign bit.
The bits above it encode either the integer *value* itself if non-negative,
or (&minus;*value*&minus;1) otherwise.
Zero becomes 0, -1 becomes 1, 1 becomes 2, -2 becomes 3, and so on.

ZigZag encoding is well-suited to CBE
because it encodes both small positive and small negative numbers
in the range -64 and +63 as 7-bit unsigned integers from 0 to 127,
which CBE in turn encodes in a single byte with no header overhead.


### String blobs

[UTF-8](https://en.wikipedia.org/wiki/UTF-8)
has become the dominant code for serializing
internationalized character strings into byte sequences.
Because of this,
it is natural and recommended that blobs containing strings
be encoded using UTF-8,
unless there is a particular reason to do otherwise
and the format in which the strings are embedded
has established a way for encoders to signal or agree on
another character encoding.

A string consisting of only a single character
from the US-ASCII character set (Unicode/UCS character codes 0-127)
will get encoded not only by UTF-8,
but also by the delimiting blob encoding,
as the identical byte value between 0 and 127.
Thus, blob encoding not only ensures that ASCII text
embedded in binary data is not unnecessarily obfuscated
and can readily be found by file scanning tools and such,
but also that single-character strings occupy only one byte.

A UTF-8 string requiring multiple bytes to encode &ndash;
whether because it contains multiple characters
or because it encodes a single non-ASCII character &ndash;
get blob-encoded with only one additional byte of embedding overhead
provided the string's UTF-8 serialization is less than 64 bytes long,
or two bytes of overhead for strings up to 16447 bytes.


### Typed data encoded with type-value pairs

While CBE itself encodes no data type information,
CBE may be used as a primitive in a data format that does so.
If a data format defines a particular set of type codes as integers,
for example,
then it is easy to represent a typed data value as
a *pair* of consecutive CBE-encoded byte sequences:
the first containing the integer type code for the data
contained in the second byte sequence.
Assuming the most commonly-used types are assigned codes from 0 to 127,
these types will CBE-encode to a single byte,
without restricting the size or extensibility of the type code space.

Alternatively, if the data format wishes types to be human-readable strings,
such as
[URI schemes](https://en.wikipedia.org/wiki/Uniform_Resource_Identifier) or
[media types](https://en.wikipedia.org/wiki/Media_type),
then the first sequence in the pair can instead be
a CBE-encoded UTF-8 string denoting the type.
If compactness is desired despite the use of strings as type names,
then the most commonly-used types may be assigned shorthands
comprised of a single 7-bit ASCII character,
again ensuring that these types encode to only one byte.


### Maps of key-value pairs

A related already-common practice is to encode
a *maps*, *dictionaries*, or *objects* as a sequence of key-value pairs.
Each key may be a string,
as in [JSON](https://en.wikipedia.org/wiki/JSON)
or [XML](https://en.wikipedia.org/wiki/XML),
or an integer,
as in [protobufs](https://developers.google.com/protocol-buffers).
Using CBE-encoded pairs for this purpose makes it easy 
for both the key and the value part of each pair
to be arbitrary-length in principle
while optimizing common cases like small-integer or single-character keys.

The entire even-length sequence of CBE-encoded blobs
may then be CBE-encoded in turn
to bind together all the pairs comprising a particular map structure
and embed it in other structures,
including other maps.


