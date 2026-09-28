# appleoracle

macOS's own `libcompression`, exposed so a test can **ask the platform what the
bytes should be** instead of asserting against bytes the implementation under
test produced.

Nothing here is a codec.

```go
enc, ok := appleoracle.Encode(appleoracle.LZFSE, src)
got, ok := appleoracle.Decode(appleoracle.LZFSE, enc, len(src))
```

## Why

One organisation answered that question three different ways:

| repository | how it asks Apple |
|---|---|
| `go-compressions/lzfse` | links `libcompression` and compares both directions |
| `go-compressions/adc` | commits files `hdiutil` produced |
| `go-compressions/lz4` | **builds Apple-format frames by hand** — no oracle at all |

The first is what caught the short-literal-payload defect in `lzfse#12`: 48 of
64 Apple-encoded runs the decoder refused. The third is the one that worries:
a fixture the implementation's author wrote can be wrong in exactly the way the
implementation is wrong, and agree with it.

A committed witness is real, and it goes stale without saying so. A live oracle
cannot.

## Availability

Everything is behind `darwin && cgo`. Elsewhere the package is a stub:
`Available()` reports false and both entry points **refuse** rather than
returning a plausible-looking empty slice, so a consumer can import it
unconditionally from a `_test.go` and skip.

## Failure is loud

`compression_encode_buffer` reports failure by writing zero bytes — which is
also what it writes for an input it declines to compress, and for an empty
input. Those are not the same thing, so:

- `Encode` returns `ok == false` for "the platform produced no stream". It
  cannot say why, and the doc comment says so. **LZVN declines a 1-byte input**,
  measured. A caller reads that as *the oracle has nothing to say here*.
- an empty input is `(nil, true)`, because nothing to compress is not a refusal.
- `Decode` decodes into `want+1` bytes, not `want`. `compression_decode_buffer`
  fills the buffer it is given, so decoding into exactly `want` returns `want`
  for a stream holding **more** and the truncation is invisible. Measured:
  asking for `len(src)-1` succeeded and handed back a prefix.

## Scope

`LZFSE`, `LZVN` and `LZ4` — what a repository here compares against today.
`LZ4` is Apple's **framed** variant (`bv41`/`bv4-`/`bv4$`), not a bare block,
and a test pins that; picking `COMPRESSION_LZ4_RAW` instead would give an
oracle that disagrees with a correct implementation on every input.

The list stops there. An entry nobody calls is an entry nobody has run, and a
wrong constant produces an oracle that confidently disagrees with everything.
ADC is absent because `libcompression` does not implement it — which is why
`go-compressions/adc` uses committed `hdiutil` output and will keep doing so.
