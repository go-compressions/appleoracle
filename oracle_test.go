package appleoracle_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/go-compressions/appleoracle"
)

func sample(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	b := make([]byte, n)
	phrase := []byte("the quick brown fox jumps over the lazy dog. ")
	for i := range b {
		if r.Intn(3) == 0 {
			b[i] = byte(r.Intn(256))
		} else {
			b[i] = phrase[i%len(phrase)]
		}
	}
	return b
}

var algos = []appleoracle.Algorithm{appleoracle.LZFSE, appleoracle.LZVN, appleoracle.LZ4}

func TestTheOracleRoundTrips(t *testing.T) {
	if !appleoracle.Available() {
		t.Skip("no libcompression on this build")
	}
	// Sizes that cross Apple's 4 KiB block boundary, where LZFSE switches
	// between the LZVN and bvx2 paths.
	sizes := []int{1, 100, 4095, 4096, 4097, 65536}
	for _, a := range algos {
		served, declined := 0, []int{}
		for _, n := range sizes {
			src := sample(n, int64(n))
			enc, ok := appleoracle.Encode(a, src)
			if !ok {
				// Not a breakage: the platform declines to compress inputs it
				// cannot shrink, and says so the same way it says "failed".
				// The floor below is what keeps this from swallowing a real one.
				declined = append(declined, n)
				continue
			}
			got, ok := appleoracle.Decode(a, enc, n)
			if !ok {
				t.Fatalf("%v: the platform could not read back its own %d-byte stream", a, n)
			}
			if !bytes.Equal(got, src) {
				t.Errorf("%v: round trip changed %d bytes", a, n)
			}
			served++
		}
		for _, n := range declined {
			if n >= 4096 {
				t.Errorf("%v: declined %d bytes -- that is the regime these codecs are judged in, "+
					"so the oracle is not merely quiet, it is missing", a, n)
			}
			t.Logf("%v: declined %d bytes (too small to compress)", a, n)
		}
		// A population floor. A sweep that silently served nothing reports no
		// failures either, and this test would then be indistinguishable from
		// its own absence.
		if served < 3 {
			t.Errorf("%v: only %d of %d sizes round-tripped", a, served, len(sizes))
		}
	}
}

// TestTheOracleCanTellWhenTheBytesAreWrong is the control. An oracle that
// accepts anything proves nothing about the codec it is judging, so it has to
// be shown refusing a stream it should refuse.
func TestTheOracleCanTellWhenTheBytesAreWrong(t *testing.T) {
	if !appleoracle.Available() {
		t.Skip("no libcompression on this build")
	}
	src := sample(8192, 7)
	for _, a := range algos {
		enc, ok := appleoracle.Encode(a, src)
		if !ok || len(enc) < 32 {
			t.Fatalf("%v: no usable stream to damage", a)
		}
		damaged := append([]byte(nil), enc...)
		// Past any header, inside the payload.
		damaged[len(damaged)/2] ^= 0xff
		got, ok := appleoracle.Decode(a, damaged, len(src))
		if ok && bytes.Equal(got, src) {
			t.Errorf("%v: a flipped byte decoded back to the original -- "+
				"this oracle cannot distinguish right bytes from wrong ones", a)
		}
	}
}

// TestLZ4IsApplesFramedVariant pins the constant that go-compressions/lz4
// needs. COMPRESSION_LZ4 is the "bv41"/"bv4-"/"bv4$" block sequence, and
// COMPRESSION_LZ4_RAW is a bare block; picking the wrong one gives an oracle
// that disagrees with a correct implementation on every input.
func TestLZ4IsApplesFramedVariant(t *testing.T) {
	if !appleoracle.Available() {
		t.Skip("no libcompression on this build")
	}
	enc, ok := appleoracle.Encode(appleoracle.LZ4, sample(16384, 3))
	if !ok {
		t.Fatal("Encode refused")
	}
	if len(enc) < 4 || !bytes.HasPrefix(enc, []byte("bv4")) {
		t.Errorf("stream does not start with an Apple block magic: %q", enc[:min(8, len(enc))])
	}
	if !bytes.HasSuffix(enc, []byte("bv4$")) {
		t.Errorf("stream does not end with the bv4$ marker: %q", enc[max(0, len(enc)-8):])
	}
}

// TestAnEmptyInputIsNotAFailure: libcompression signals failure by writing
// zero bytes, which is the same thing it writes for an input that legitimately
// produces nothing. A caller must not have to special-case that.
func TestAnEmptyInputIsNotAFailure(t *testing.T) {
	if !appleoracle.Available() {
		t.Skip("no libcompression on this build")
	}
	for _, a := range algos {
		if _, ok := appleoracle.Encode(a, nil); !ok {
			t.Errorf("%v: Encode(nil) reported failure", a)
		}
		if _, ok := appleoracle.Decode(a, nil, 0); !ok {
			t.Errorf("%v: Decode(nil, 0) reported failure", a)
		}
	}
}

// TestDecodeRefusesAShortResult: a decoder that produced fewer bytes than the
// caller knows the input held has not decoded it, and a test that accepted the
// prefix would be asserting on half an answer.
func TestDecodeRefusesAShortResult(t *testing.T) {
	if !appleoracle.Available() {
		t.Skip("no libcompression on this build")
	}
	src := sample(4096, 11)
	enc, ok := appleoracle.Encode(appleoracle.LZFSE, src)
	if !ok {
		t.Fatal("Encode refused")
	}
	if _, ok := appleoracle.Decode(appleoracle.LZFSE, enc, len(src)-1); ok {
		t.Error("Decode accepted a length it could not have produced")
	}
}

// TestAnUnknownAlgorithmRefuses: a bad constant must not silently become some
// other codec.
func TestAnUnknownAlgorithmRefuses(t *testing.T) {
	if !appleoracle.Available() {
		t.Skip("no libcompression on this build")
	}
	if _, ok := appleoracle.Encode(appleoracle.Algorithm(99), sample(1024, 5)); ok {
		t.Error("Encode accepted an algorithm that does not exist")
	}
}

// TestTheStubRefusesRatherThanReturningNothing: on a build without
// libcompression the entry points must fail, not hand back an empty slice that
// a forgetful caller would compare against.
func TestTheStubRefusesRatherThanReturningNothing(t *testing.T) {
	if appleoracle.Available() {
		t.Skip("this build has libcompression")
	}
	if _, ok := appleoracle.Encode(appleoracle.LZFSE, []byte("x")); ok {
		t.Error("Encode reported success without libcompression")
	}
	if _, ok := appleoracle.Decode(appleoracle.LZFSE, []byte("x"), 1); ok {
		t.Error("Decode reported success without libcompression")
	}
}

func TestAlgorithmString(t *testing.T) {
	for _, c := range []struct {
		a    appleoracle.Algorithm
		want string
	}{
		{appleoracle.LZFSE, "LZFSE"},
		{appleoracle.LZVN, "LZVN"},
		{appleoracle.LZ4, "LZ4"},
		{appleoracle.Algorithm(99), "Algorithm(?)"},
	} {
		if got := c.a.String(); got != c.want {
			t.Errorf("String() = %q, want %q", got, c.want)
		}
	}
}
