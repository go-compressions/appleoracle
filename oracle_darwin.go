//go:build darwin && cgo

package appleoracle

/*
#cgo LDFLAGS: -lcompression
#include <stdlib.h>
#include <compression.h>

// COMPRESSION_LZVN is not in the public header on every SDK, so it goes in by
// value. 0x900 is the constant Apple's own tooling uses and the one
// go-compressions/lzfse has been measuring against.
static size_t oracle_encode(int algo, unsigned char *dst, size_t dstcap,
                            const unsigned char *src, size_t n) {
    return compression_encode_buffer(dst, dstcap, src, n, NULL,
                                     (compression_algorithm)algo);
}
static size_t oracle_decode(int algo, unsigned char *dst, size_t dstcap,
                            const unsigned char *src, size_t n) {
    return compression_decode_buffer(dst, dstcap, src, n, NULL,
                                     (compression_algorithm)algo);
}
*/
import "C"

import "unsafe"

// Available reports whether this build can ask the platform. True here, false
// in the stub, so a _test.go can import this package unconditionally.
func Available() bool { return true }

func raw(a Algorithm) C.int {
	switch a {
	case LZFSE:
		return C.COMPRESSION_LZFSE
	case LZVN:
		return 0x900
	case LZ4:
		return C.COMPRESSION_LZ4
	}
	// An unknown Algorithm must not become some other codec by accident: 0 is
	// not a valid compression_algorithm, so libcompression refuses it and the
	// caller gets ok == false rather than a confident wrong answer.
	return 0
}

// Encode compresses src with the platform's implementation of a.
//
// ok is false when the platform produced no stream. compression_encode_buffer
// signals that with a zero return and does not say why, so this cannot tell an
// unsupported algorithm from an input the codec declined to compress -- and it
// DOES decline: LZVN returns nothing for a 1-byte input, measured. A caller
// therefore reads ok == false as "the oracle has nothing to say about this
// input", skips it, and must not read it as "the oracle is broken".
//
// An empty input is the exception: nothing to compress is not a refusal, so
// that case is (nil, true).
func Encode(a Algorithm, src []byte) (out []byte, ok bool) {
	// Apple's own guidance for a worst case; a stream that does not fit is a
	// refusal rather than a truncation, and the caller sees ok == false.
	dst := make([]byte, len(src)+4096)
	n := C.oracle_encode(raw(a), (*C.uchar)(unsafe.Pointer(&dst[0])), C.size_t(len(dst)),
		ptr(src), C.size_t(len(src)))
	if n == 0 {
		return nil, len(src) == 0
	}
	return dst[:n], true
}

// Decode decompresses src with the platform's implementation of a, expecting
// exactly want bytes.
//
// ok is false when the platform refused the stream OR produced a different
// length than want -- in either direction. Those are one answer on purpose: a
// result that is not exactly what the caller knows the input held is not a
// decode, and a test that accepted it would be asserting on a prefix.
//
// ⛔ The destination is want+1 bytes, not want. compression_decode_buffer fills
// the buffer it is given and reports how much it wrote, so decoding into
// exactly want bytes returns want for a stream that holds MORE than want and
// the truncation is invisible. Measured: asking for len(src)-1 succeeded and
// handed back a prefix. One spare byte makes an over-long stream report
// want+1, which is not want, which is a refusal.
func Decode(a Algorithm, src []byte, want int) (out []byte, ok bool) {
	if want == 0 {
		// Nothing to write, so a zero return says nothing about failure.
		return nil, true
	}
	dst := make([]byte, want+1)
	n := C.oracle_decode(raw(a), (*C.uchar)(unsafe.Pointer(&dst[0])), C.size_t(len(dst)),
		ptr(src), C.size_t(len(src)))
	if int(n) != want {
		return nil, false
	}
	return dst[:n], true
}

// ptr is nil-safe: &src[0] panics on an empty slice, and an empty input is a
// case every codec has to get right.
func ptr(b []byte) *C.uchar {
	if len(b) == 0 {
		return nil
	}
	return (*C.uchar)(unsafe.Pointer(&b[0]))
}
