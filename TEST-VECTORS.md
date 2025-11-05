# Test Vectors for Go OPRF Implementation

This document contains test vectors extracted from the C implementation to verify byte-for-byte compatibility.

## Source
These vectors are from the IRTF CFRG OPRF specification test vectors, used in liboprf's test suite.
Source: `~/projects/git/liboprf/src/tests/testvecs2h.py`

## Configuration
- **Mode**: 0 (Base OPRF mode)
- **Group**: ristretto255
- **Hash**: SHA-512
- **Identifier**: "ristretto255-SHA512"
- **Seed**: `a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3`
- **Server Private Key (skSm)**: `5ebcea5ee37023ccb9fc2d2019f9d7737be85591ae8652ffa9ef0f4d37063b0e`

## Test Case 0: Single Byte Input

### Input
- **Input (hex)**: `00`
- **Input (description)**: Single zero byte

### Blinding Phase (Client)
- **Blind (r) (hex)**: `64d37aed22a27f5191de1c1d69fadb899d8862b58eb4220029e036ec4c1f6706`
- **Blinded Element (alpha) (hex)**: `609a0ae68c15a3cf6903766461307e5c8bb2f95e7e6550e1ffa2dc99e412803c`

### Evaluation Phase (Server)
- **Evaluation Element (beta) (hex)**: `7ec6578ae5120958eb2db1745758ff379e77cb64fe77b0b2d8cc917ea0869c7e`

### Final Output (Client)
- **Output (hex)**: `527759c3d9366f277d8c6020418d96bb393ba2afb20ff90df23fb7708264e2f3ab9135e3bd69955851de4b1f9fe8a0973396719b7912ba9ee8aa7d0b5e24bcf6`

### Test Flow
```
Input: 00
  ↓ [oprf_Blind with r]
Blinded Element: 609a0ae68c15a3cf6903766461307e5c8bb2f95e7e6550e1ffa2dc99e412803c
  ↓ [oprf_Evaluate with skSm]
Evaluation Element: 7ec6578ae5120958eb2db1745758ff379e77cb64fe77b0b2d8cc917ea0869c7e
  ↓ [oprf_Unblind with r]
Unblinded Element: N
  ↓ [oprf_Finalize with input]
Output: 527759c3d9366f277d8c6020418d96bb393ba2afb20ff90df23fb7708264e2f3ab9135e3bd69955851de4b1f9fe8a0973396719b7912ba9ee8aa7d0b5e24bcf6
```

## Test Case 1: Repeated Byte Pattern

### Input
- **Input (hex)**: `5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a`
- **Input (description)**: 17 bytes of 0x5a

### Blinding Phase (Client)
- **Blind (r) (hex)**: `64d37aed22a27f5191de1c1d69fadb899d8862b58eb4220029e036ec4c1f6706`
- **Blinded Element (alpha) (hex)**: `da27ef466870f5f15296299850aa088629945a17d1f5b7f5ff043f76b3c06418`

### Evaluation Phase (Server)
- **Evaluation Element (beta) (hex)**: `b4cbf5a4f1eeda5a63ce7b77c7d23f461db3fcab0dd28e4e17cecb5c90d02c25`

### Final Output (Client)
- **Output (hex)**: `f4a74c9c592497375e796aa837e907b1a045d34306a749db9f34221f7e750cb4f2a6413a6bf6fa5e19ba6348eb673934a722a7ede2e7621306d18951e7cf2c73`

### Test Flow
```
Input: 5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a
  ↓ [oprf_Blind with r]
Blinded Element: da27ef466870f5f15296299850aa088629945a17d1f5b7f5ff043f76b3c06418
  ↓ [oprf_Evaluate with skSm]
Evaluation Element: b4cbf5a4f1eeda5a63ce7b77c7d23f461db3fcab0dd28e4e17cecb5c90d02c25
  ↓ [oprf_Unblind with r]
Unblinded Element: N
  ↓ [oprf_Finalize with input]
Output: f4a74c9c592497375e796aa837e907b1a045d34306a749db9f34221f7e750cb4f2a6413a6bf6fa5e19ba6348eb673934a722a7ede2e7621306d18951e7cf2c73
```

## Domain Separation Tags

From the test vectors and C implementation:

- **Hash-to-Group DST**: `HashToGroup-OPRFV1-\x00-ristretto255-SHA512`
  - Hex: `48617368546f47726f75702d4f50524656312d002d72697374726574746f3235352d534841353132`
- **Finalize DST**: `Finalize`

## Notes for Implementation

1. **Deterministic Blinding**: Both test cases use the same blind value `r`. In production, `r` must be randomly generated, but for testing we use the fixed value to match expected outputs.

2. **Byte Compatibility**: The Go implementation must produce **exactly** these hex values to be compatible with the C implementation.

3. **Test Vector Validation**: Each function should be tested independently:
   - `oprf_Blind(input, r)` → should produce expected BlindedElement
   - `oprf_Evaluate(skSm, BlindedElement)` → should produce expected EvaluationElement
   - `oprf_Unblind(r, EvaluationElement)` → should produce intermediate N
   - `oprf_Finalize(input, N)` → should produce expected Output

4. **End-to-End Test**: Full OPRF flow from input to output should match expected Output.

## Additional Test Vectors Needed

For comprehensive testing, we should also extract:
- [ ] Hash-to-group test vectors (expand_message_xmd intermediate values)
- [ ] Threshold OPRF test vectors (from toprf.c test)
- [ ] Edge cases (empty input, maximum length input)
- [ ] Invalid point detection tests

## Cleanup Notice

This file should be deleted once all test vectors are incorporated into Go test files (`oprf/oprf_test.go`).
