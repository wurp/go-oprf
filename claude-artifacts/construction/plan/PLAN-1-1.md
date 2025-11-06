# PLAN-1-1: Project Setup and C Code Analysis

## Goal
Set up the Go project structure and thoroughly understand the C implementation before beginning translation.

## Prerequisites
- C source code at ~/projects/git/liboprf
- Go installed and working

## Detailed Tasks

### Task 1.1: Initialize Go Module
**Action:**
- Run `go mod init github.com/wurp/go-oprf`
- Add dependencies:
  - `github.com/gtank/ristretto255`
  - `golang.org/x/crypto`
- Verify dependencies download correctly

**Success Criteria:**
- go.mod and go.sum files exist
- `go mod tidy` completes without errors

**Time Estimate:** 15 minutes

### Task 1.2: Create Package Structure
**Action:**
- Create directories:
  - `oprf/` - basic OPRF implementation
  - `toprf/` - threshold OPRF implementation
  - `internal/utils/` - shared utilities
- Create placeholder files with package declarations

**Success Criteria:**
- All directories exist
- Each directory has a .go file with proper package declaration
- `go build ./...` completes without errors

**Time Estimate:** 15 minutes

### Task 1.3: Study C Header Files
**Action:**
- Read ~/projects/git/liboprf/README.md
- Read ~/projects/git/liboprf/src/oprf.h
- Read ~/projects/git/liboprf/src/toprf.h
- Document:
  - Public API functions and their signatures
  - Key data structures
  - Constants and their values
  - Function dependencies

**Success Criteria:**
- Create C-API-ANALYSIS.md with:
  - All public functions listed
  - Data structures documented
  - Constants documented
  - High-level flow understood

**Time Estimate:** 1-2 hours

### Task 1.4: Study C Implementation
**Action:**
- Read ~/projects/git/liboprf/src/oprf.c in detail
- Identify key algorithms and logic flow
- Note any libsodium-specific calls
- Document non-obvious implementation details

**Success Criteria:**
- C-API-ANALYSIS.md updated with:
  - Function-by-function notes
  - Libsodium calls mapped to Go equivalents
  - Complex logic explained
  - Note any constant-time operations

**Time Estimate:** 2-3 hours

### Task 1.5: Extract Test Vectors
**Action:**
- Locate C test files in ~/projects/git/liboprf/src/tests/
- Read test.c and related test files
- Extract test vectors (inputs, keys, expected outputs)
- Document in TEST-VECTORS.md

**Success Criteria:**
- TEST-VECTORS.md contains:
  - At least 3-5 test vectors for basic OPRF
  - Inputs, keys, and expected outputs in hex format
  - Test descriptions

**Time Estimate:** 1-2 hours

### Task 1.6: Create Initial Test Files
**Action:**
- Create oprf/oprf_test.go with test vector structure
- Create helper functions for hex encoding/decoding
- Add placeholder tests (will fail initially)

**Success Criteria:**
- oprf_test.go exists with test structure
- Tests compile but skip or fail (expected at this stage)

**Time Estimate:** 30 minutes

## Total Estimated Time
5-8 hours (approximately 1 work day)

## Notes
- Focus on understanding before coding
- Document everything - future self will thank you
- Test vectors are critical for verification
- Take extra time on constant-time operations

## Completion Criteria
- [x] Go module initialized with dependencies
- [x] Package structure created
- [x] C-API-ANALYSIS.md complete
- [x] TEST-VECTORS.md complete with at least 3-5 vectors
- [x] Initial test files created
- [x] Ready to begin actual porting in PLAN-1-2

## Dependencies for Next Step
PLAN-1-2 (Basic OPRF Implementation) will need:
- All test vectors from this step
- Understanding of C API from analysis
- Package structure in place
