# C API Analysis for liboprf

This document contains analysis of the C implementation from ~/projects/git/liboprf to guide the Go port.

## Overview

liboprf implements OPRF(ristretto255, SHA-512) following RFC 9497 and RFC 9380. The library provides:
- Basic OPRF functions
- Threshold OPRF (TOPRF) using 3hashTDH protocol
- DKG (Distributed Key Generation) protocols

## Constants

### Basic OPRF Constants
- `OPRF_BYTES`: 64 (output size of OPRF)
- `crypto_core_ristretto255_BYTES`: 32 (ristretto255 point size)
- `crypto_core_ristretto255_SCALARBYTES`: 32 (scalar size)
- `crypto_core_ristretto255_HASHBYTES`: 64 (hash-to-curve input size)
- `crypto_hash_sha512_BYTES`: 64 (SHA-512 output size)

### Domain Separation Tags
- `VOPRF`: "OPRFV1"
- Hash-to-group DST: "HashToGroup-OPRFV1-\x00-ristretto255-SHA512"
- Finalize DST: "Finalize"

## Basic OPRF API (oprf.h/oprf.c)

### 1. oprf_KeyGen
```c
void oprf_KeyGen(uint8_t kU[crypto_core_ristretto255_SCALARBYTES]);
```
**Purpose**: Generate OPRF private key
**Implementation**: Uses `crypto_core_ristretto255_scalar_random()` from libsodium
**Go Mapping**: Use ristretto255 library's scalar random generation

### 2. oprf_Blind
```c
int oprf_Blind(const uint8_t *x, const uint16_t x_len,
               uint8_t r[crypto_core_ristretto255_SCALARBYTES],
               uint8_t alpha[crypto_core_ristretto255_BYTES]);
```
**Purpose**: Blind input x with random scalar r
**Steps**:
1. Hash input x to curve point H0 using `voprf_hash_to_group()`
2. Generate random scalar r
3. Compute alpha = H0^r (scalar multiplication)

**Returns**: 0 on success, -1 on error

### 3. oprf_Evaluate
```c
int oprf_Evaluate(const uint8_t k[crypto_core_ristretto255_SCALARBYTES],
                  const uint8_t alpha[crypto_core_ristretto255_BYTES],
                  uint8_t beta[crypto_core_ristretto255_BYTES]);
```
**Purpose**: Server evaluates blinded input with private key
**Implementation**: Simple scalar multiplication: beta = alpha^k
**Note**: This is straightforward - just a single `crypto_scalarmult_ristretto255()` call

### 4. oprf_Unblind
```c
int oprf_Unblind(const uint8_t r[crypto_core_ristretto255_SCALARBYTES],
                 const uint8_t beta[crypto_core_ristretto255_BYTES],
                 uint8_t N[crypto_core_ristretto255_BYTES]);
```
**Purpose**: Remove blinding factor r from evaluated result
**Steps**:
1. Validate beta is a valid group element
2. Compute inverse of r: ir = 1/r
3. Compute N = beta^ir (scalar multiplication)

**Returns**: 0 on success, -1 on error
**Critical**: Must validate point before unblinding

### 5. oprf_Finalize
```c
int oprf_Finalize(const uint8_t *x, const uint16_t x_len,
                  const uint8_t N[crypto_core_ristretto255_BYTES],
                  uint8_t rwdU[OPRF_BYTES]);
```
**Purpose**: Compute final OPRF output
**Implementation**: Hash of concatenated inputs with length prefixes
**Steps**:
1. Hash(htons(len(x)) || x || htons(len(N)) || N || "Finalize")
2. Uses SHA-512
3. Output is 64 bytes

**Note**: Uses network byte order (htons) for length prefixes

## Hash-to-Curve Functions

### voprf_hash_to_group
```c
int voprf_hash_to_group(const uint8_t *msg, const uint16_t msg_len,
                        uint8_t p[crypto_core_ristretto255_BYTES]);
```
**Purpose**: Hash arbitrary input to ristretto255 point
**Implementation**: Following RFC 9380
**Steps**:
1. Call `oprf_expand_message_xmd()` with DST="HashToGroup-OPRFV1-\x00-ristretto255-SHA512"
2. Get 64 uniform bytes
3. Map to curve using `crypto_core_ristretto255_from_hash()`

### oprf_expand_message_xmd
```c
int oprf_expand_message_xmd(const uint8_t *msg, const uint16_t msg_len,
                            const uint8_t *dst, const uint8_t dst_len,
                            const uint8_t len_in_bytes,
                            uint8_t *uniform_bytes);
```
**Purpose**: Expand message to uniform bytes using XMD (expand_message_xmd from RFC 9380)
**Parameters**:
- H = SHA-512
- b_in_bytes = 64 (output size of SHA-512)
- r_in_bytes = 128 (input block size of SHA-512)

**Algorithm**:
1. ell = ceil(len_in_bytes / 64)
2. Abort if ell > 255
3. DST_prime = DST || I2OSP(len(DST), 1)
4. Z_pad = I2OSP(0, 128) [all zeros]
5. l_i_b_str = I2OSP(len_in_bytes, 2)
6. msg_prime = Z_pad || msg || l_i_b_str || I2OSP(0, 1) || DST_prime
7. b_0 = H(msg_prime)
8. b_1 = H(b_0 || I2OSP(1, 1) || DST_prime)
9. For i in (2, ..., ell):
   - b_i = H(strxor(b_0, b_(i-1)) || I2OSP(i, 1) || DST_prime)
10. uniform_bytes = b_1 || ... || b_ell
11. Return substr(uniform_bytes, 0, len_in_bytes)

**Note**: Implementation unrolls loop by 2 for optimization

## Threshold OPRF API (toprf.h)

### Data Structures

#### TOPRF_Share
```c
typedef struct {
  uint8_t index;
  uint8_t value[crypto_core_ristretto255_SCALARBYTES];
} __attribute((packed)) TOPRF_Share;
```
- `TOPRF_Share_BYTES`: 33 bytes (1 + 32)
- `TOPRF_Part_BYTES`: 33 bytes (32 + 1)

### Key Functions

#### toprf_create_shares
```c
void toprf_create_shares(const uint8_t secret[32],
                         const uint8_t n,
                         const uint8_t threshold,
                         uint8_t shares[n][TOPRF_Share_BYTES]);
```
**Purpose**: Shamir secret sharing over ristretto255
**Note**: Creates n shares where threshold shares can reconstruct secret

#### toprf_Evaluate
```c
int toprf_Evaluate(const uint8_t k[TOPRF_Share_BYTES],
                   const uint8_t blinded[32],
                   const uint8_t self, const uint8_t *indexes,
                   const uint16_t index_len,
                   uint8_t Z[TOPRF_Part_BYTES]);
```
**Purpose**: Threshold version of oprf_Evaluate
**Note**: Requires knowing all participating peer indices in advance for Lagrange coefficient computation

#### toprf_thresholdcombine
```c
int toprf_thresholdcombine(const size_t response_len,
                           const uint8_t _responses[response_len][TOPRF_Part_BYTES],
                           uint8_t result[32]);
```
**Purpose**: Combine threshold partial evaluations
**Implementation**: Uses Lagrange interpolation in the exponent

#### toprf_3hashtdh
```c
int toprf_3hashtdh(const uint8_t k[TOPRF_Share_BYTES],
                   const uint8_t z[TOPRF_Share_BYTES],
                   const uint8_t alpha[32],
                   const uint8_t *ssid_S, const uint16_t ssid_S_len,
                   uint8_t beta[TOPRF_Part_BYTES]);
```
**Purpose**: Implements 3hashTDH protocol from Gu et al. 2024
**Parameters**:
- k: share of secret key
- z: random zero-sharing (for threshold security)
- alpha: blinded element from client
- ssid_S: session-specific identifier (must be same for all participants)

## Libsodium to Go Mappings

### Scalar Operations
| Libsodium Function | Go Equivalent (ristretto255) |
|--------------------|------------------------------|
| `crypto_core_ristretto255_scalar_random()` | `NewScalar().Random()` |
| `crypto_core_ristretto255_scalar_invert()` | `Invert()` on Scalar |
| `crypto_core_ristretto255_scalar_add()` | `Add()` on Scalar |
| `crypto_core_ristretto255_scalar_sub()` | `Subtract()` on Scalar |
| `crypto_core_ristretto255_scalar_mul()` | `Multiply()` on Scalar |

### Point Operations
| Libsodium Function | Go Equivalent (ristretto255) |
|--------------------|------------------------------|
| `crypto_scalarmult_ristretto255()` | `ScalarMult()` on Element |
| `crypto_core_ristretto255_from_hash()` | `FromUniformBytes()` on Element |
| `crypto_core_ristretto255_is_valid_point()` | Always valid after decode or compute |
| `crypto_core_ristretto255_add()` | `Add()` on Element |

### Hash Operations
| Libsodium Function | Go Equivalent |
|--------------------|---------------|
| `crypto_hash_sha512()` | `sha512.Sum512()` |
| `crypto_hash_sha512_init/update/final()` | `sha512.New()` and `Write()` |

## Implementation Notes

### Memory Security
- C code uses `sodium_mlock()` and `sodium_munlock()` for sensitive data
- Go doesn't have equivalent; rely on Go's memory management
- Consider zeroing sensitive byte slices when done

### Byte Order
- Length prefixes use network byte order (big-endian)
- Use `binary.BigEndian.PutUint16()` in Go

### Error Handling
- C functions return 0 for success, -1 for error
- Go should use idiomatic error returns

### Constant-Time Operations
- Original uses libsodium's constant-time operations
- ristretto255 library provides constant-time scalar operations
- Critical for security: all scalar operations must be constant-time

### Test Vectors
- C code has conditional compilation for test vectors (`CFRG_TEST_VEC`)
- Need to extract actual test vector values from C test files
- Vectors should test each function independently and end-to-end

## Dependencies for Go Implementation

### Required Packages
1. `github.com/gtank/ristretto255` - ristretto255 curve operations
2. `crypto/sha512` - SHA-512 hashing
3. `encoding/binary` - byte order conversions

### Optional (for testing)
1. `encoding/hex` - hex encoding/decoding for test vectors

## Port Priority

### Phase 1: Basic OPRF (Step 2 of PLAN-1)
1. oprf_expand_message_xmd
2. voprf_hash_to_group
3. oprf_Blind
4. oprf_Evaluate
5. oprf_Unblind
6. oprf_Finalize
7. oprf_KeyGen

### Phase 2: Threshold OPRF (Step 3 of PLAN-1)
1. Lagrange interpolation functions
2. toprf_create_shares
3. toprf_Evaluate
4. toprf_thresholdcombine
5. toprf_3hashtdh

## Critical Implementation Details

### Hash-to-Curve Domain Separation
The DST (Domain Separation Tag) is critical for security:
- Basic hash-to-group: "HashToGroup-OPRFV1-\x00-ristretto255-SHA512"
- Must exactly match C implementation for compatibility

### Finalize Format
The finalize function has a specific format:
```
hash(
  htons(len(x)) ||
  x ||
  htons(len(N)) ||
  N ||
  "Finalize"
)
```
Must match exactly for byte compatibility.

### Scalar Inversion
The `oprf_Unblind` function requires scalar inversion (computing 1/r).
This must be done in constant time to avoid timing attacks.

## Temporary File Notice
This file should be archived or deleted once:
- All functions are ported to Go
- All tests pass with C test vectors
- Implementation is verified as byte-compatible
