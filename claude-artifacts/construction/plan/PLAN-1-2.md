# PLAN-1-2: Basic OPRF Implementation

## Goal
Implement core OPRF functions in Go with byte-for-byte compatibility with the C implementation.

## Prerequisites
- Step 1 complete (project setup, C analysis, test vectors)
- C-API-ANALYSIS.md available for reference
- TEST-VECTORS.md available with test cases
- Test structure in oprf_test.go ready

## Detailed Tasks

### Task 2.1: Implement expand_message_xmd
**Action:**
- Implement `expandMessageXMD()` function per RFC 9380
- Parameters: SHA-512 (b_in_bytes=64, r_in_bytes=128)
- Algorithm steps from C-API-ANALYSIS.md:
  - Handle DST_prime construction
  - Implement Z_pad, l_i_b_str
  - Compute b_0, b_1, ..., b_ell with proper XOR
- Add test vectors for intermediate values
- Critical: must match C implementation exactly

**Success Criteria:**
- Function compiles and runs
- Test with known inputs produces expected outputs
- Code includes detailed comments explaining RFC 9380 steps

**Time Estimate:** 1-1.5 hours

### Task 2.2: Implement hash_to_group
**Action:**
- Implement `hashToGroup()` function
- Call expandMessageXMD with DST="HashToGroup-OPRFV1-\x00-ristretto255-SHA512"
- Get 64 uniform bytes
- Map to ristretto255 using `FromUniformBytes()`
- Add test for hash-to-group operation

**Success Criteria:**
- Function compiles and runs
- Produces valid ristretto255 points
- Test verifies correct DST usage
- Documented with RFC reference

**Time Estimate:** 45 minutes

### Task 2.3: Implement Blind
**Action:**
- Implement `Blind(input []byte, blind []byte) (r, alpha []byte, err error)`
- Steps:
  1. Hash input to curve point using hashToGroup
  2. Use provided blind as scalar r (for testing) or generate random (for production)
  3. Compute alpha = H0 * r (scalar multiplication)
- Return both r and alpha

**Success Criteria:**
- Function signature matches test expectations
- Test with test vectors: TestBlind passes
- Properly handles both fixed and random blind values
- Error handling for invalid inputs

**Time Estimate:** 1 hour

**Test Update:** Remove `t.Skip()` from TestBlind, verify test passes

### Task 2.4: Implement Evaluate
**Action:**
- Implement `Evaluate(k []byte, alpha []byte) (beta []byte, err error)`
- Simple scalar multiplication: beta = alpha^k
- Validate alpha is valid ristretto255 point

**Success Criteria:**
- Function compiles and runs
- TestEvaluate passes with test vectors
- Proper error handling for invalid points

**Time Estimate:** 30 minutes

**Test Update:** Remove `t.Skip()` from TestEvaluate, verify test passes

### Task 2.5: Implement Unblind
**Action:**
- Implement `Unblind(r []byte, beta []byte) (n []byte, err error)`
- Steps:
  1. Validate beta is valid ristretto255 point
  2. Compute r_inv = 1/r (scalar inversion)
  3. Compute n = beta^r_inv
- Must use constant-time operations

**Success Criteria:**
- Function compiles and runs
- TestUnblind passes
- Uses constant-time scalar inversion
- Proper validation of beta

**Time Estimate:** 45 minutes

**Test Update:** Remove `t.Skip()` from TestUnblind, verify test passes

### Task 2.6: Implement Finalize
**Action:**
- Implement `Finalize(input []byte, n []byte) (output []byte, err error)`
- Format: hash(htons(len(input)) || input || htons(len(n)) || n || "Finalize")
- Use binary.BigEndian.PutUint16 for length prefixes
- SHA-512 for hashing
- Output is 64 bytes

**Success Criteria:**
- Function compiles and runs
- TestFinalize passes with test vectors
- Exact byte-for-byte match with C implementation
- Length prefixes in network byte order

**Time Estimate:** 45 minutes

**Test Update:** Remove `t.Skip()` from TestFinalize, verify test passes

### Task 2.7: Implement KeyGen
**Action:**
- Implement `KeyGen() ([]byte, error)`
- Generate random ristretto255 scalar
- Return 32-byte scalar

**Success Criteria:**
- Function compiles and runs
- Generates valid scalars
- Uses crypto-secure randomness
- Add basic test for KeyGen

**Time Estimate:** 20 minutes

### Task 2.8: End-to-End Integration Test
**Action:**
- Remove `t.Skip()` from TestOPRFEndToEnd
- Verify complete flow: Blind → Evaluate → Unblind → Finalize
- Ensure test vectors produce exact expected outputs
- Run all tests: `go test -v ./oprf/...`

**Success Criteria:**
- All tests pass without skips
- End-to-end test produces byte-perfect outputs
- Test coverage is comprehensive
- No race conditions (run with `-race`)

**Time Estimate:** 30 minutes

### Task 2.9: Add Constants and Documentation
**Action:**
- Add package-level constants:
  - OPRF_BYTES = 64
  - ScalarBytes = 32
  - ElementBytes = 32
  - HashToGroupDST
  - FinalizeDST
- Add comprehensive package documentation
- Document all exported functions with Go doc comments
- Add usage examples in comments

**Success Criteria:**
- All constants defined and used
- `go doc` shows clear documentation
- Functions have proper doc comments
- Examples are clear and helpful

**Time Estimate:** 45 minutes

### Task 2.10: Code Review and Cleanup
**Action:**
- Review all code for:
  - Proper error handling
  - Security considerations (constant-time operations)
  - Code comments and clarity
  - Consistent style
- Run `go fmt ./...`
- Run `go vet ./...`
- Check for any TODO comments to address
- Verify no debug code remains

**Success Criteria:**
- Code passes go fmt and go vet
- No security issues identified
- All TODOs resolved or documented for later
- Code is production-ready

**Time Estimate:** 30 minutes

## Total Estimated Time
6.5-7.5 hours (approximately 1 work day)

## Implementation Order
Functions depend on each other, so implement in this order:
1. expandMessageXMD (foundation for hash-to-group)
2. hashToGroup (foundation for Blind)
3. Blind (client-side function)
4. Evaluate (server-side function)
5. Unblind (client-side function)
6. Finalize (client-side function)
7. KeyGen (standalone utility)
8. Integration testing
9. Documentation and cleanup

## Critical Success Factors

### Byte Compatibility
Every function must produce **exactly** the same bytes as C implementation:
- Test vectors must pass byte-for-byte
- Any deviation indicates implementation error

### Security Considerations
- Scalar operations must be constant-time
- Point validation before operations
- Proper error handling without leaking information
- No timing side-channels

### Testing Strategy
- Test each function independently with test vectors
- Test end-to-end flow
- Test error conditions
- Test with invalid inputs

## Completion Criteria
- [x] expandMessageXMD implemented and tested
- [x] hashToGroup implemented and tested
- [x] Blind implemented and tested
- [x] Evaluate implemented and tested
- [x] Unblind implemented and tested
- [x] Finalize implemented and tested
- [x] KeyGen implemented and tested
- [x] All unit tests passing (no t.Skip())
- [x] TestOPRFEndToEnd passing with exact test vector matches
- [x] Code documented with Go doc comments
- [x] Code passes go fmt, go vet
- [x] Security review: constant-time operations verified
- [x] Ready to move to Step 3 (Threshold OPRF)

## Dependencies for Next Step
PLAN-1-3 (Threshold OPRF Implementation) will need:
- All basic OPRF functions working and tested
- Understanding of point/scalar operations from this step
- Test framework extended for threshold operations

## Notes
- Keep C-API-ANALYSIS.md open for reference throughout
- Commit working code after each major function is complete
- Run tests frequently during development
- If any test vector fails, debug before moving forward
- Focus on correctness over optimization initially
