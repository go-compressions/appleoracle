// Package appleoracle exposes macOS's own libcompression so a test can ask the
// platform what the bytes should be, instead of asserting against bytes the
// implementation under test produced.
//
// It exists because one fleet answered that question three different ways:
//
//   - go-compressions/lzfse links libcompression directly and compares both
//     directions against it. That is what caught the short-literal-payload
//     defect in #12 -- 48 of 64 Apple-encoded runs the decoder refused.
//   - go-compressions/adc commits files hdiutil produced. A committed witness
//     is real, and it goes stale without saying so.
//   - go-compressions/lz4 builds Apple-format frames BY HAND in its tests,
//     with no oracle at all -- so those tests can be wrong in exactly the way
//     the implementation is wrong, and agree with it.
//
// Every function here is a thin call into compression_encode_buffer /
// compression_decode_buffer. Nothing in this package is a codec.
//
// # Availability
//
// The whole package is behind `darwin && cgo`. On every other build it is
// empty, and [Available] reports false, so a consumer can import it
// unconditionally from a _test.go file and skip.
//
// # Failure is loud
//
// libcompression reports failure by returning zero bytes written, which is
// indistinguishable from "the empty input compressed to nothing". Every
// function here returns an explicit ok, and a caller that ignores it has
// written a test that passes when the oracle is not working -- which is the
// failure mode this package exists to remove, not to reproduce.
package appleoracle

// Algorithm is one of the codecs libcompression implements.
//
// The list stops at what a repository in this organisation actually compares
// against today. An entry nobody calls is an entry nobody has run, and a
// wrong constant here produces an oracle that confidently disagrees with
// everything.
type Algorithm int

const (
	// LZFSE is Apple's LZFSE (COMPRESSION_LZFSE), the raw bvx stream.
	// Caller: go-compressions/lzfse.
	LZFSE Algorithm = iota
	// LZVN is the LZVN stream Apple uses for small payloads
	// (COMPRESSION_LZVN, 0x900). Caller: go-compressions/lzfse.
	LZVN
	// LZ4 is Apple's FRAMED LZ4 (COMPRESSION_LZ4): the "bv41"/"bv4-"/"bv4$"
	// block sequence, not a bare LZ4 block. Caller: go-compressions/lz4,
	// whose apple.go implements exactly that framing.
	LZ4
)

func (a Algorithm) String() string {
	switch a {
	case LZFSE:
		return "LZFSE"
	case LZVN:
		return "LZVN"
	case LZ4:
		return "LZ4"
	}
	return "Algorithm(?)"
}
