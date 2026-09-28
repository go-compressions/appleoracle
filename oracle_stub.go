//go:build !(darwin && cgo)

package appleoracle

// The platform is not here to be asked. Available() is false and the two
// entry points refuse rather than returning a plausible-looking nothing: a
// test that forgets to check Available() must fail, not quietly compare
// against an empty slice.

// Available reports whether this build can ask the platform.
func Available() bool { return false }

// Encode always refuses on a build without libcompression.
func Encode(Algorithm, []byte) (out []byte, ok bool) { return nil, false }

// Decode always refuses on a build without libcompression.
func Decode(Algorithm, []byte, int) (out []byte, ok bool) { return nil, false }
