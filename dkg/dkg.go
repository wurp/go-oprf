// Package dkg implements Distributed Key Generation (DKG) for threshold
// cryptography.
//
// DKG allows multiple participants to collaboratively generate a shared secret
// key in a distributed manner, where no single party ever knows the complete
// key. The secret is split using Shamir secret sharing such that any threshold
// number of participants can use their shares to perform cryptographic
// operations, but fewer participants learn nothing.
//
// # Protocol Overview
//
// The DKG protocol consists of three phases:
//
//  1. Start: Each participant generates a random polynomial and creates:
//     - Commitments to the polynomial coefficients (broadcast to all)
//     - Secret shares for each participant (sent privately)
//
//  2. Verify: Each participant verifies that received shares match the
//     sender's commitments using VerifyCommitment()
//
//  3. Finish: Each participant combines all received shares to compute
//     their final secret share using Finish()
//
// # Usage Example
//
//	// Example: 3 participants with threshold 2
//	n, threshold := uint8(3), uint8(2)
//
//	// Phase 1: Each participant starts DKG
//	commitments1, shares1, _ := dkg.Start(n, threshold)
//	commitments2, shares2, _ := dkg.Start(n, threshold)
//	commitments3, shares3, _ := dkg.Start(n, threshold)
//
//	// Participants exchange commitments (broadcast) and shares (private channels)
//	// Participant 1 receives shares: shares1[0], shares2[0], shares3[0]
//	// Participant 2 receives shares: shares1[1], shares2[1], shares3[1]
//	// etc.
//
//	// Phase 2: Each participant verifies received shares
//	allCommitments := [][]*ristretto255.Element{commitments1, commitments2, commitments3}
//	receivedShares := []toprf.Share{shares1[0], shares2[0], shares3[0]}
//	fails, _ := dkg.VerifyCommitments(n, threshold, 1, allCommitments, receivedShares)
//	if len(fails) > 0 {
//	    // Handle verification failures
//	}
//
//	// Phase 3: Combine shares to get final secret share
//	finalShare, _ := dkg.Finish(receivedShares, 1)
//
//	// Now participant 1 has their share of the distributed secret
//	// Any threshold participants can collaborate to use the secret
//
// # Security Properties
//
// - Secret never exists in one location
// - Threshold participants needed to reconstruct or use secret
// - Commitment scheme prevents malicious share dealing
// - Compatible with threshold OPRF operations
//
// # Verifiable Secret Sharing (VSS)
//
// This package also provides VSS primitives in vss.go for enhanced security:
//   - Pedersen commitments for information-theoretic hiding
//   - Public verifiability of shares
//   - Robustness against malicious participants
//
// See the vss.go file for VSS-specific functions.
//
// # Compatibility
//
// Ported from liboprf's dkg.c and fully compatible with the C implementation.
package dkg

import (
	"crypto/subtle"
	"errors"

	"github.com/gtank/ristretto255"
	"github.com/wurp/go-oprf/toprf"
)

// Start initializes the DKG protocol for one participant.
// Generates polynomial coefficients, commitments, and shares for all participants.
//
// Parameters:
//   - n: number of participants
//   - threshold: minimum shares needed to use the key (must be 1 < threshold <= n)
//
// Returns:
//   - commitments: threshold commitments to polynomial coefficients (broadcast to all)
//   - shares: n shares, one for each participant (send privately)
//
// Corresponds to dkg_start() in dkg.c:70-102
func Start(n, threshold uint8) (
	commitments []*ristretto255.Element,
	shares []toprf.Share,
	err error,
) {
	if threshold < 2 || threshold > n {
		return nil, nil, errors.New("dkg: threshold must be > 1 and <= n")
	}

	// Generate random polynomial coefficients
	a := make([]*ristretto255.Scalar, threshold)
	for k := uint8(0); k < threshold; k++ {
		a[k], err = randomScalar()
		if err != nil {
			return nil, nil, err
		}
	}

	// Compute commitments to coefficients: C_k = g^a_k
	commitments = make([]*ristretto255.Element, threshold)
	for k := uint8(0); k < threshold; k++ {
		commitments[k] = ristretto255.NewElement().ScalarBaseMult(a[k])
	}

	// Create shares for each participant: s_j = f(j)
	shares = make([]toprf.Share, n)
	for j := uint8(1); j <= n; j++ {
		shares[j-1] = polynom(j, threshold, a)
	}

	return commitments, shares, nil
}

// VerifyCommitment verifies that a share from peer i matches its commitments.
// This checks that the share is consistent with the polynomial that peer i committed to.
//
// Parameters:
//   - n: number of participants
//   - threshold: the threshold parameter
//   - self: index of current participant (1-based)
//   - i: index of peer being verified (1-based)
//   - commitments: the threshold commitments from peer i
//   - share: the share received from peer i
//
// Returns error if verification fails.
//
// Corresponds to dkg_verify_commitment() in dkg.c:104-149
func VerifyCommitment(n, threshold, self, i uint8, commitments []*ristretto255.Element, share toprf.Share) error {
	if i == self {
		return nil // Don't verify our own share
	}

	// v0 = g^(share.value)
	v0 := ristretto255.NewElement().ScalarBaseMult(share.Value)

	// v1 = C[0] * C[1]^j * C[2]^j^2 * ... * C[threshold-1]^j^(threshold-1)
	// where j = self
	j := scalarFromUint8(self)

	// Start with v1 = C[0]
	v1 := ristretto255.NewElement()
	v1.Decode(commitments[0].Encode(nil))

	// Add terms C[k]^j^k for k=1..threshold-1
	for k := uint8(1); k < threshold; k++ {
		// Compute j^k
		jPowK := scalarFromUint8(1)
		for exp := uint8(0); exp < k; exp++ {
			jPowK.Multiply(jPowK, j)
		}

		// tmP = C[k]^j^k
		tmP := ristretto255.NewElement()
		tmP.ScalarMult(jPowK, commitments[k])

		// v1 = v1 + tmP
		v1.Add(v1, tmP)
	}

	// Check v0 == v1
	v0Bytes := v0.Encode(nil)
	v1Bytes := v1.Encode(nil)
	if subtle.ConstantTimeCompare(v0Bytes, v1Bytes) != 1 {
		return errors.New("dkg: commitment verification failed")
	}

	return nil
}

// VerifyCommitments verifies shares from all peers.
//
// Parameters:
//   - n: number of participants
//   - threshold: the threshold parameter
//   - self: index of current participant (1-based)
//   - commitments: commitments from all n peers
//   - shares: shares received from all n peers
//
// Returns list of peer indices that failed verification.
//
// Corresponds to dkg_verify_commitments() in dkg.c:151-169
func VerifyCommitments(n, threshold, self uint8, commitments [][]*ristretto255.Element, shares []toprf.Share) ([]uint8, error) {
	var fails []uint8

	for i := uint8(1); i <= n; i++ {
		if i == self {
			continue
		}

		err := VerifyCommitment(n, threshold, self, i, commitments[i-1], shares[i-1])
		if err != nil {
			fails = append(fails, i)
		}
	}

	return fails, nil
}

// Finish combines shares from all participants to compute the final secret share.
// All shares must have the same index (self).
//
// Parameters:
//   - shares: n shares addressed to this participant (one from each peer)
//   - self: index of current participant
//
// Returns the final secret share for this participant.
//
// Corresponds to dkg_finish() in dkg.c:171-186
func Finish(shares []toprf.Share, self uint8) (toprf.Share, error) {
	result := ristretto255.NewScalar()
	result.Decode([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) // = 0

	for i := range shares {
		if shares[i].Index != self {
			return toprf.Share{}, errors.New("dkg: share has incorrect index")
		}
		result.Add(result, shares[i].Value)
	}

	return toprf.Share{
		Index: self,
		Value: result,
	}, nil
}

// Reconstruct recovers the group secret from threshold or more shares.
// Uses Lagrange interpolation at x=0 to recover the constant term (the secret).
//
// Parameters:
//   - shares: at least threshold shares from different participants
//
// Returns the reconstructed group secret.
//
// Corresponds to dkg_reconstruct() in dkg.c:188-204
func Reconstruct(shares []toprf.Share) (*ristretto255.Scalar, error) {
	if len(shares) == 0 {
		return nil, errors.New("dkg: no shares provided")
	}

	// Interpolate at x=0 to get the secret (constant term)
	secret, err := toprf.InterpolateScalar(0, shares)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// scalarFromUint8 creates a ristretto255 scalar from a uint8 value.
// Used for index arithmetic in commitment verification.
func scalarFromUint8(v uint8) *ristretto255.Scalar {
	var buf [32]byte
	buf[0] = v
	s := ristretto255.NewScalar()
	s.Decode(buf[:])
	return s
}
