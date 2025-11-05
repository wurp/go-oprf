# Go OPRF - Porting Instructions

## Overview

Port the liboprf C library to pure Go, focusing on the core OPRF and threshold OPRF (3hashTDH) functionality needed for password-authenticated key exchange.

**Source repository**: `~/projects/git/liboprf`

**Goal**: Create a pure Go implementation that:
- Implements the same cryptographic protocols as liboprf
- Produces byte-compatible output (verifiable via test vectors)
- Works identically across all platforms (Windows, Linux, Mac, iOS, Android)
- Avoids CGo entirely

## Porting Philosophy

### Priorities (in order)

1. **Correctness**: Output must match C implementation exactly
2. **Alignment with C code**: Structure and logic should mirror the C implementation
3. **Go idioms**: Use Go conventions only when they don't conflict with #2
4. **Performance**: Optimize only after correctness is proven

### When to Diverge from C

**Use Go conventions for**:
- Error handling (return `error` instead of int error codes)
- Memory management (let GC handle it; no manual malloc/free)
- Package structure and naming
- Documentation (use Go doc comments)

**Stay close to C for**:
- Algorithm structure and flow
- Variable names (translate but keep recognizable)
- Function signatures (translate types but keep logic identical)
- Constant-time operations (preserve timing characteristics)

## Source Code Structure

### Core Files to Port (in order)

The C implementation in `~/projects/git/liboprf/src/`:

1. **oprf.c / oprf.h** (~570 lines)
   - Basic OPRF implementation
   - OPRF(ristretto255, SHA-512) per IRTF CFRG spec
   - Port first, get working and tested

2. **toprf.c / toprf.h** (~480 lines)
   - Threshold OPRF implementation
   - Includes 3hashTDH protocol from Gu et al. 2024
   - Port second, after basic OPRF working

3. **Supporting files as needed**:
   - dkg.c/h: Distributed Key Generation (if needed for your use case)
   - utils.c/h: Utility functions

Total core code: ~1,000-1,500 lines

### Files to Skip (initially)

- toprf-update.c/h: Key rotation protocol (can add later)
- mpmult.c/h: Multi-party multiplication (used by updates)
- stp-dkg.c/h: Semi-trusted DKG variant (use simpler DKG first)
- jni.c: Java bindings (not needed)
- noise_xk/: Noise protocol (not part of core OPRF)

## Dependencies

### C Dependencies (what liboprf uses)

- **libsodium**: Provides ristretto255, SHA-512, random bytes, secure memory

### Go Replacements (use these)

1. **Ristretto255**:
   - `github.com/gtank/ristretto255` - Recommended by filippo.io/edwards25519
   - Well-established, implements ristretto255 prime-order group
   - Compatible with libsodium's ristretto255

2. **Hashing**:
   - `crypto/sha512` - Standard library
   - `golang.org/x/crypto/sha3` - If SHA3 needed

3. **Random bytes**:
   - `crypto/rand` - Standard library, cryptographically secure

4. **Big integers** (if needed):
   - `math/big` - Standard library
   - Use sparingly; prefer fixed-size byte arrays when possible

5. **Key derivation**:
   - `golang.org/x/crypto/hkdf` - Standard extended library

### Installation

```bash
cd ~/projects/go-oprf
go mod init github.com/wurp/go-oprf  # Or your repo path
go get github.com/gtank/ristretto255
go get golang.org/x/crypto
```

## Porting Process

### Phase 1: Setup and Basic OPRF

1. **Study the C implementation**:
   ```bash
   cd ~/projects/git/liboprf
   # Read the README
   cat README.md

   # Study the header files first (they define the API)
   cat src/oprf.h
   cat src/toprf.h

   # Then the implementation
   cat src/oprf.c
   ```

2. **Create Go package structure**:
   ```
   go-oprf/
     oprf/
       oprf.go          # Basic OPRF
       oprf_test.go     # Tests with C test vectors
     toprf/
       toprf.go         # Threshold OPRF
       toprf_test.go
     internal/
       utils/           # Internal utilities
   ```

3. **Port basic OPRF first** (`oprf.c` → `oprf/oprf.go`):
   - Translate data structures
   - Translate functions one by one
   - Keep same logic flow
   - Add test for each function

4. **Verify against C implementation**:
   - Extract test vectors from C tests (`src/tests/test.c`)
   - Run same inputs through both implementations
   - Compare outputs byte-for-byte
   - Any difference = bug in port

### Phase 2: Threshold OPRF

5. **Port threshold OPRF** (`toprf.c` → `toprf/toprf.go`):
   - Same process as basic OPRF
   - Build on verified basic OPRF
   - Focus on 3hashTDH protocol

6. **Verify threshold functionality**:
   - Test with threshold parameters (M-of-N)
   - Verify all threshold combinations work
   - Compare with C test vectors

### Phase 3: Integration

7. **Add DKG if needed**:
   - Port dkg.c for distributed key generation
   - Test key generation protocol

8. **Performance testing**:
   - Benchmark against requirements
   - Identify any performance issues
   - Optimize only if necessary (and only constant-time operations)

## Type Translations

### C to Go Type Mapping

```c
// C                          // Go
unsigned char[32]       →     [32]byte
uint8_t                 →     uint8
uint16_t                →     uint16
uint32_t                →     uint32
uint64_t                →     uint64
size_t                  →     int or uint (be careful with platform differences)
void*                   →     []byte or specific type
int (return code)       →     error

// Structures
typedef struct { ... }  →     type StructName struct { ... }

// Arrays
uint8_t buf[32]        →     var buf [32]byte
uint8_t* buf           →     buf []byte  (or [32]byte if fixed size known)
```

### Example Translation

**C code** (from oprf.c):
```c
int oprf_Evaluate(const uint8_t *pwdU, const size_t pwdU_len,
                  const uint8_t skS[OPRF_BYTES],
                  uint8_t output[OPRF_BYTES]) {
    // ... implementation
    return 0;  // success
}
```

**Go code**:
```go
// Evaluate computes the OPRF output for the given password and server key.
func Evaluate(pwdU []byte, skS [OPRFBytes]byte) ([OPRFBytes]byte, error) {
    var output [OPRFBytes]byte
    // ... implementation (keep same logic as C)
    return output, nil
}
```

## Security Considerations

### Critical: Preserve Constant-Time Operations

Cryptographic code must avoid timing side-channels.

**In C**, libsodium provides constant-time functions:
- `sodium_memcmp()` - constant-time comparison
- `sodium_increment()` - constant-time increment

**In Go**, use:
- `crypto/subtle.ConstantTimeCompare()` - for comparisons
- Be aware: Go compiler may optimize away constant-time properties
- Review generated assembly for critical operations

### Memory Safety

**Advantages of Go**:
- No buffer overflows (bounds checking)
- No use-after-free
- No double-free
- Automatic memory management

**Watch out for**:
- Timing side-channels (Go may not be constant-time where C was)
- Keeping secrets in memory longer than necessary
- Go heap vs stack allocation differences

### Clearing Sensitive Data

**C approach**:
```c
sodium_memzero(secret, sizeof(secret));
```

**Go approach**:
```go
// Clear sensitive data when done
for i := range secret {
    secret[i] = 0
}
// Note: Go GC may copy data before clearing; less secure than C
```

## Testing Strategy

### Test Vector Approach

1. **Extract C test vectors**:
   - Look in `~/projects/git/liboprf/src/tests/`
   - Find input/output pairs
   - Document in Go test files

2. **Create test file format**:
   ```go
   type TestVector struct {
       Name     string
       Input    []byte
       Key      [32]byte
       Expected [32]byte
   }

   var basicOPRFVectors = []TestVector{
       {
           Name:     "test vector 1 from C implementation",
           Input:    decodeHex("..."),
           Key:      decodeHex("..."),
           Expected: decodeHex("..."),
       },
       // ... more vectors
   }
   ```

3. **Test every function**:
   ```go
   func TestEvaluate(t *testing.T) {
       for _, tv := range basicOPRFVectors {
           t.Run(tv.Name, func(t *testing.T) {
               result, err := Evaluate(tv.Input, tv.Key)
               if err != nil {
                   t.Fatalf("Evaluate failed: %v", err)
               }
               if !bytes.Equal(result[:], tv.Expected[:]) {
                   t.Errorf("Output mismatch\nGot:  %x\nWant: %x", result, tv.Expected)
               }
           })
       }
   }
   ```

### Cross-Implementation Testing

Ideally, generate test vectors from running C implementation:

```bash
# In C code directory
cd ~/projects/git/liboprf/src
# Build and run tests with verbose output to capture test vectors
make test
```

Then use those exact vectors in Go tests.

## Documentation

### Code Comments

Keep comments aligned with C implementation:
- If C code has a comment explaining logic, port that comment
- Add additional Go-specific comments where needed
- Reference the C source location: `// From oprf.c:123`

### Function Documentation

Use Go doc format:

```go
// Evaluate computes the OPRF output for the given password.
//
// This implements the server-side evaluation in the OPRF protocol
// as specified in IRTF CFRG OPRF(ristretto255, SHA-512).
//
// Corresponds to oprf_Evaluate() in liboprf's oprf.c.
func Evaluate(pwdU []byte, skS [OPRFBytes]byte) ([OPRFBytes]byte, error)
```

## References

### Specifications

- **IRTF CFRG OPRF**: https://datatracker.ietf.org/doc/draft-irtf-cfrg-voprf/
- **RFC 9497** (OPRF using Prime-Order Groups): https://www.rfc-editor.org/rfc/rfc9497
- **Gu et al. 2024**: "Threshold PAKE with Security against Compromise of all Servers"
  - ePrint: https://eprint.iacr.org/2024/1455
  - Implements 3hashTDH protocol

### Source Code

- **liboprf C implementation**: `~/projects/git/liboprf`
- **ristretto255 Go**: https://github.com/gtank/ristretto255

### Background Reading

If you need to understand the cryptography:
- Study the IRTF CFRG OPRF specification (it's readable)
- Read the ristretto255 documentation
- The C code is well-structured; use it as the primary reference

## Development Workflow

### Recommended Approach

1. **Port incrementally**: One function at a time
2. **Test immediately**: Write test before moving to next function
3. **Verify constantly**: Run all tests after each change
4. **Commit often**: Each working function is a commit
5. **Document decisions**: Note any places where you diverged from C and why

### Example Workflow for One Function

```bash
# 1. Read C function
cat ~/projects/git/liboprf/src/oprf.c  # Find the function

# 2. Port to Go
vim oprf/oprf.go  # Translate function

# 3. Write test
vim oprf/oprf_test.go  # Add test with vector from C

# 4. Test
go test ./oprf -v

# 5. Fix until test passes

# 6. Commit
git add .
git commit -m "Port oprf.Evaluate() function"

# 7. Move to next function
```

## Success Criteria

Port is complete when:

- [ ] All core OPRF functions ported from oprf.c
- [ ] All threshold OPRF functions ported from toprf.c
- [ ] All test vectors from C implementation pass in Go
- [ ] Cross-platform tests pass (run on Linux, Mac, Windows minimum)
- [ ] Code review by cryptography-aware developer
- [ ] No CGo dependencies
- [ ] Documentation complete

## Getting Started

```bash
cd ~/projects/go-oprf

# Initialize Go module
go mod init github.com/wurp/go-oprf  # Adjust path as needed

# Get dependencies
go get github.com/gtank/ristretto255
go get golang.org/x/crypto

# Create package structure
mkdir -p oprf toprf internal/utils

# Start reading the C code
cd ~/projects/git/liboprf
cat README.md
cat src/oprf.h

# Begin porting
cd ~/projects/go-oprf
# Create oprf/oprf.go and start translating...
```

## Questions or Issues

If you encounter:

- **Unclear C code**: Check the test files in `~/projects/git/liboprf/src/tests/`
- **Cryptographic questions**: Refer to IRTF CFRG OPRF spec or ePrint papers
- **Type conversion issues**: Document in code comments, maintain correctness
- **Performance issues**: Measure first, optimize only if needed
- **Test failures**: Cross-reference with C implementation byte-by-byte

## Notes for Maintainer

This port is for security-critical code (password authentication). Key considerations:

1. **Constant-time operations**: Critical for preventing timing attacks
2. **Test vector compatibility**: Essential for proving correctness
3. **Platform independence**: Must work identically on all platforms
4. **No shortcuts**: Better to take time and get it right than rush and introduce vulnerabilities

The C implementation is well-regarded and funded by European Commission. Staying close to it reduces risk of introducing bugs during porting.
