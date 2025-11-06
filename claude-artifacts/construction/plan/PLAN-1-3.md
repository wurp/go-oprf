# PLAN-1-3: Threshold OPRF Implementation

## Goal
Implement threshold OPRF functions in Go with byte-for-byte compatibility with the C implementation, including Lagrange interpolation and the 3HashTDH protocol.

## Prerequisites
- Step 2 complete (basic OPRF implementation working)
- C-API-ANALYSIS.md available for reference
- oprf package functions tested and working
- Understanding of Lagrange interpolation and Shamir secret sharing

## Detailed Tasks

### Task 3.1: Study C Threshold Implementation
**Action:**
- Read ~/projects/git/liboprf/src/toprf.c in detail
- Understand Lagrange interpolation functions (interpolate, lcoeff, coeff)
- Understand Shamir secret sharing implementation
- Study 3HashTDH protocol implementation
- Review test files at ~/projects/git/liboprf/src/tests/toprf.c
- Document any additional implementation details in C-API-ANALYSIS.md

**Success Criteria:**
- Complete understanding of toprf.c algorithms
- Note any edge cases or special handling
- Document zero-sharing concept for 3HashTDH
- Identify all test vectors needed

**Time Estimate:** 1.5-2 hours

### Task 3.2: Extract Threshold Test Vectors
**Action:**
- Read test files in ~/projects/git/liboprf/src/tests/toprf.c
- Extract test vectors for:
  - Lagrange coefficient computation
  - Secret sharing (create_shares)
  - Threshold evaluation
  - Threshold combine
  - 3HashTDH protocol
- Document in TEST-VECTORS.md with inputs, keys, and expected outputs

**Success Criteria:**
- TEST-VECTORS.md updated with threshold test vectors
- At least 3-5 test cases covering different scenarios
- Test vectors include all intermediate values when possible

**Time Estimate:** 1-1.5 hours

### Task 3.3: Implement Data Structures
**Action:**
- Create Share struct in toprf/toprf.go:
  ```go
  type Share struct {
      Index uint8
      Value [32]byte
  }
  ```
- Add package-level constants:
  - ShareBytes = 33 (1 byte index + 32 byte value)
  - PartBytes = 33 (32 byte element + 1 byte index)
- Add utility functions for encoding/decoding shares

**Success Criteria:**
- Share struct defined
- Constants defined
- Helper functions for serialization/deserialization
- Basic tests for struct operations

**Time Estimate:** 30 minutes

### Task 3.4: Implement Lagrange Coefficient Functions
**Action:**
- Implement `lcoeff(index, x, degree, peers) scalar`:
  - Computes Lagrange coefficient for f(x)
  - Used in polynomial interpolation
- Implement `coeff(index, peers) scalar`:
  - Wrapper for lcoeff with x=0 (for f(0))
  - Used in threshold combine operations
- Use constant-time scalar operations from ristretto255

**Success Criteria:**
- Both functions compile and run
- Test with known coefficients
- Proper error handling for invalid inputs
- Code documented with algorithm explanation

**Time Estimate:** 1-1.5 hours

### Task 3.5: Implement Polynomial Interpolation
**Action:**
- Implement `interpolate(x, shares) scalar`:
  - Uses Lagrange interpolation to compute f(x)
  - Takes array of Share structs
  - Returns scalar value
- Algorithm:
  1. For each share, compute Lagrange coefficient
  2. Multiply coefficient by share value
  3. Sum all results

**Success Criteria:**
- Function compiles and runs
- Test with known polynomial evaluations
- Handles edge cases (minimum shares, etc.)
- Documented with mathematical explanation

**Time Estimate:** 1 hour

### Task 3.6: Implement Secret Sharing
**Action:**
- Implement `CreateShares(secret, n, threshold) []Share`:
  - Creates n shares using Shamir's secret sharing
  - Threshold shares can reconstruct secret
  - Random polynomial coefficients
- Algorithm:
  1. Secret is a₀ (constant term)
  2. Generate random scalars a₁, a₂, ..., a_{threshold-1}
  3. For each i in 1..n: shares[i] = f(i) where f(x) = a₀ + a₁x + a₂x² + ...
- Use crypto-secure randomness

**Success Criteria:**
- Function compiles and runs
- Test: reconstruct secret from threshold shares
- Test: threshold-1 shares cannot reconstruct
- Proper validation of inputs (n >= threshold, etc.)

**Time Estimate:** 1-1.5 hours

### Task 3.7: Implement Threshold Evaluate
**Action:**
- Implement `Evaluate(k Share, blinded []byte, self uint8, indexes []uint8) ([]byte, error)`:
  - Threshold version of oprf.Evaluate
  - Computes Lagrange coefficient for self based on indexes
  - Returns element with index: (beta || self_index)
- Algorithm:
  1. Validate blinded is valid ristretto255 point
  2. Compute Lagrange coefficient c = coeff(self, indexes)
  3. Compute share_scalar = k.Value * c
  4. Compute beta = blinded^share_scalar
  5. Return beta || self_index (33 bytes)

**Success Criteria:**
- Function compiles and runs
- Test with test vectors
- Proper validation of inputs
- Documented with protocol explanation

**Time Estimate:** 1 hour

### Task 3.8: Implement Threshold Combine
**Action:**
- Implement `ThresholdCombine(responses [][]byte) ([]byte, error)`:
  - Combines partial evaluations from threshold peers
  - Uses Lagrange interpolation in the exponent
  - Returns final element (unblinded N)
- Algorithm:
  1. Parse each response into (element, index)
  2. For each response, compute Lagrange coefficient
  3. Compute element^coefficient for each
  4. Multiply all results together
  5. Return final element

**Success Criteria:**
- Function compiles and runs
- Test end-to-end: CreateShares → Evaluate (multiple) → Combine
- Produces same result as non-threshold OPRF
- Proper error handling

**Time Estimate:** 1-1.5 hours

### Task 3.9: Implement 3HashTDH Protocol
**Action:**
- Implement `ThreeHashTDH(k Share, z Share, alpha []byte, ssid []byte) ([]byte, error)`:
  - Implements 3HashTDH from Gu et al. 2024 paper
  - Provides threshold security against server compromise
- Algorithm (from C code):
  1. Compute k_scalar from k.Value
  2. Compute z_scalar from z.Value
  3. Parse alpha to point
  4. Hash ssid to scalar: h = H(ssid) mod order
  5. Compute scalar s = k + h*z
  6. Compute beta = alpha^s
  7. Return beta || k.Index

**Success Criteria:**
- Function compiles and runs
- Test with test vectors
- Verify SSID properly included in computation
- Documented with paper reference

**Time Estimate:** 1.5-2 hours

### Task 3.10: Create Comprehensive Tests
**Action:**
- Create toprf/toprf_test.go with test structure
- Add tests for:
  - TestLagrangeCoefficients
  - TestInterpolation
  - TestCreateShares
  - TestSecretReconstruction
  - TestThresholdEvaluate
  - TestThresholdCombine
  - TestThreeHashTDH
  - TestTOPRFEndToEnd
- Use test vectors from TEST-VECTORS.md
- Test edge cases and error conditions

**Success Criteria:**
- All tests compile
- All tests pass with test vectors
- Edge cases tested
- Error conditions tested
- Test coverage is comprehensive

**Time Estimate:** 2-2.5 hours

### Task 3.11: Integration Testing
**Action:**
- Create end-to-end test demonstrating full threshold OPRF flow:
  1. Client: Blind input
  2. Split server key into shares (n=5, threshold=3)
  3. Three servers: Each evaluates with their share
  4. Client: Combine responses
  5. Client: Unblind result
  6. Client: Finalize
- Verify output matches non-threshold OPRF output
- Test with multiple threshold configurations

**Success Criteria:**
- End-to-end test passes
- Output matches basic OPRF output
- Works with different (n, threshold) configurations
- Documented example code

**Time Estimate:** 1 hour

### Task 3.12: Add Documentation
**Action:**
- Add comprehensive package documentation to toprf package
- Document all exported functions with Go doc comments
- Add usage examples for common scenarios
- Document security considerations:
  - SSID must be same for all participants
  - Zero-sharing security properties
  - Threshold security model
- Add references to 3HashTDH paper

**Success Criteria:**
- All functions have doc comments
- `go doc toprf` shows clear documentation
- Usage examples included
- Security considerations documented

**Time Estimate:** 45 minutes

### Task 3.13: Code Review and Cleanup
**Action:**
- Review all code for:
  - Proper error handling
  - Security considerations (constant-time operations)
  - Code comments and clarity
  - Consistent style with oprf package
- Run `go fmt ./...`
- Run `go vet ./...`
- Run all tests: `go test -v ./...`
- Run with race detector: `go test -race ./...`
- Check for any TODO comments

**Success Criteria:**
- Code passes go fmt and go vet
- All tests pass
- No race conditions detected
- Code is production-ready
- Consistent with oprf package style

**Time Estimate:** 45 minutes

## Total Estimated Time
14-17 hours (approximately 2 work days)

## Implementation Order
Functions have dependencies, implement in this order:
1. Study C code and extract test vectors
2. Data structures and constants
3. Lagrange coefficient functions (foundation)
4. Polynomial interpolation
5. Secret sharing (CreateShares)
6. Threshold Evaluate
7. Threshold Combine
8. 3HashTDH protocol
9. Comprehensive testing
10. Documentation and cleanup

## Critical Success Factors

### Mathematical Correctness
- Lagrange interpolation must be exact
- Shamir secret sharing must satisfy threshold property
- Zero-sharing for 3HashTDH must preserve security

### Byte Compatibility
- All functions must produce identical bytes as C implementation
- Test vectors must pass byte-for-byte
- Encoding/decoding must match C structs

### Security Considerations
- All scalar operations must be constant-time
- Polynomial coefficients must use crypto-secure randomness
- SSID handling must prevent replay attacks
- Zero-sharing randomness must be independent

### Testing Strategy
- Test each function independently
- Test mathematical properties (e.g., threshold reconstruction)
- Test compatibility with basic OPRF functions
- Test end-to-end threshold flow
- Test error conditions and edge cases

## Completion Criteria
- [x] C implementation studied and documented
- [x] Test vectors extracted for all threshold functions
- [x] Share data structures implemented
- [x] Lagrange coefficient functions implemented and tested
- [x] Polynomial interpolation implemented and tested
- [x] CreateShares implemented and tested
- [x] Secret reconstruction verified (threshold shares work, threshold-1 don't)
- [x] Threshold Evaluate implemented and tested
- [x] Threshold Combine implemented and tested
- [x] 3HashTDH protocol implemented and tested
- [x] All unit tests passing
- [x] End-to-end integration test passing
- [x] Output matches basic OPRF for same inputs
- [x] Code documented with Go doc comments
- [x] Code passes go fmt, go vet, and race detector
- [x] Security review: constant-time operations verified
- [x] Ready to move to Step 4 (DKG or Cross-Platform Verification)

## Dependencies for Next Step
PLAN-1-4 (DKG or Cross-Platform Verification) will need:
- All threshold OPRF functions working and tested
- Understanding of distributed protocols from this step
- Decision on whether DKG is needed for use case

## Notes
- Lagrange interpolation is the mathematical foundation - get it right first
- Secret sharing and reconstruction should be tested thoroughly before moving to OPRF integration
- 3HashTDH is more complex than basic threshold - allocate sufficient time
- Keep C implementation open for reference throughout
- Commit working code after each major function
- Run tests frequently during development
- Focus on correctness over optimization initially

## References
- TOPPSS paper: https://eprint.iacr.org/2017/363 (Section 3)
- 3HashTDH paper: https://eprint.iacr.org/2024/1455
- Shamir's Secret Sharing: https://en.wikipedia.org/wiki/Shamir%27s_secret_sharing
