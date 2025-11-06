# PLAN-1-4: DKG (Distributed Key Generation) Evaluation and Implementation

## Goal
Evaluate whether Distributed Key Generation (DKG) is needed for the use case and, if required, port the appropriate DKG implementation from liboprf.

## Prerequisites
- Steps 1-3 complete (✅)
- Basic OPRF and Threshold OPRF implementations working
- All existing tests passing

## Context

### Available DKG Implementations in C
From ~/projects/git/liboprf/src/:

1. **dkg.c/h** (18KB/16KB)
   - Standard DKG implementation
   - Uses Verifiable Secret Sharing (VSS)
   - Simplest variant

2. **dkg-vss.c/h** (5.8KB/4.8KB)
   - VSS primitives used by DKG
   - Required if porting dkg.c

3. **tp-dkg.c/h** (55KB/26KB)
   - Trusted Party DKG variant
   - Larger implementation

4. **stp-dkg.c/h** (88KB/35KB)
   - Semi-Trusted Party DKG variant
   - Largest implementation
   - INSTRUCTIONS.md recommends skipping initially

### Purpose of DKG
DKG allows multiple parties to generate threshold OPRF keys without any single party knowing the complete secret key. This is essential for:
- Distributed server deployments where no single server should hold the full key
- Security against server compromise
- Threshold cryptography where M-of-N servers must cooperate

### Alternative to DKG
If the use case allows:
- Generate keys centrally and distribute shares
- Use Threshold OPRF with pre-generated keys
- This is simpler but less secure (requires trusting the key generator)

## Detailed Tasks

### Task 4.1: Evaluate Use Case Requirements
**Action:**
- Review project goal: "password-authenticated key exchange"
- Determine if distributed key generation is required
- Consult with user about deployment scenario:
  - Will keys be generated centrally or distributedly?
  - Is server compromise a threat model concern?
  - Are multiple independent servers involved?

**Success Criteria:**
- Clear decision on whether DKG is needed
- Documentation of decision rationale

**Time Estimate:** 15-30 minutes (including user consultation)

### Task 4.2a: If DKG Not Needed
**Action:**
- Document in PLAN-1.md why DKG is skipped
- Note that Threshold OPRF already supports pre-generated key shares
- Move to Step 5 (Cross-Platform Verification)

**Success Criteria:**
- PLAN-1.md updated with decision
- Ready to proceed to Step 5

**Time Estimate:** 5 minutes

### Task 4.2b: If DKG Needed - Analyze Implementation
**Action:**
- Read dkg.h to understand API
- Read dkg-vss.h for VSS primitives
- Read dkg.c to understand implementation
- Document in C-API-ANALYSIS.md:
  - DKG protocol flow
  - Data structures
  - Key functions
  - Dependencies on VSS

**Success Criteria:**
- C-API-ANALYSIS.md updated with DKG section
- Understanding of VSS + DKG relationship
- List of functions to port

**Time Estimate:** 2-3 hours

### Task 4.3: Port VSS Primitives (if DKG needed)
**Action:**
- Create dkg/vss.go package
- Port data structures from dkg-vss.h
- Port VSS functions from dkg-vss.c:
  - Secret sharing
  - Share verification
  - Commitment generation/verification
- Create dkg/vss_test.go with test vectors

**Success Criteria:**
- dkg/vss.go implemented
- VSS tests passing
- Byte-compatible with C implementation

**Time Estimate:** 4-6 hours

### Task 4.4: Port DKG Implementation (if DKG needed)
**Action:**
- Create dkg/dkg.go package
- Port data structures from dkg.h
- Port DKG protocol functions from dkg.c:
  - Key generation initialization
  - Share distribution
  - Share combination
  - Verification
- Create dkg/dkg_test.go with test vectors

**Success Criteria:**
- dkg/dkg.go implemented
- DKG tests passing
- Can generate threshold keys matching C implementation
- Integration test with toprf package

**Time Estimate:** 6-8 hours

### Task 4.5: Integration Testing (if DKG needed)
**Action:**
- Create test that uses DKG to generate keys
- Use generated keys with Threshold OPRF
- Verify entire flow works end-to-end
- Compare with C implementation results

**Success Criteria:**
- End-to-end test: DKG → Threshold OPRF → Verification
- All tests passing
- Byte-compatible output

**Time Estimate:** 2-3 hours

## Decision Tree

```
Is DKG needed for your use case?
│
├─ NO → Document decision
│        Update PLAN-1.md
│        Move to Step 5
│        (5 minutes)
│
└─ YES → Port dkg-vss.c/h (4-6 hours)
         Port dkg.c/h (6-8 hours)
         Integration testing (2-3 hours)
         Total: 12-17 hours (~2-3 days)
```

## Total Estimated Time
- **If DKG not needed**: 30 minutes
- **If DKG needed**: 14-20 hours (~2-3 work days)

## Completion Criteria

### If DKG Not Needed:
- [ ] Decision documented in PLAN-1.md
- [ ] Rationale clear
- [ ] Ready for Step 5

### If DKG Needed:
- [x] C-API-ANALYSIS.md updated with DKG section
- [x] dkg/vss.go implemented and tested
- [x] dkg/dkg.go implemented and tested
- [x] Integration test with toprf passing
- [x] All core DKG functions ported (dkg-vss.c + dkg.c lines 1-204)
- [x] Ready for Step 5

## Implementation Summary

Successfully ported core DKG functionality:
- **dkg/vss.go**: VSS primitives with Pedersen commitments (151 lines from C)
- **dkg/dkg.go**: Core DKG protocol functions (204 lines from C)
- **Tests**: Comprehensive tests including integration with Threshold OPRF
- **Excluded**: Protocol infrastructure (Noise_XK, message framing) - can add later if needed

All tests passing. DKG can generate distributed threshold keys that work with Threshold OPRF.

## Dependencies for Next Step
PLAN-1-5 (Cross-Platform Verification) will need:
- All previous implementations complete
- All tests passing
- Optional: DKG if needed for use case

## Notes
- INSTRUCTIONS.md says "if needed for your use case"
- DKG is substantial code (~24KB C code for basic version)
- Can always add DKG later if requirements change
- Threshold OPRF works with or without DKG
