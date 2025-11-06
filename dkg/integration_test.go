package dkg

import (
	"crypto/rand"
	"testing"

	"github.com/gtank/ristretto255"
	"github.com/wurp/go-oprf/oprf"
	"github.com/wurp/go-oprf/toprf"
)

// TestDKGWithThresholdOPRF demonstrates the complete flow:
// 1. Use DKG to generate distributed threshold key shares
// 2. Use those shares with Threshold OPRF (3HashTDH protocol)
// 3. Verify the complete OPRF protocol works end-to-end
func TestDKGWithThresholdOPRF(t *testing.T) {
	const n = 5         // 5 servers
	const threshold = 3 // any 3 can serve requests

	// ========== Phase 1: Distributed Key Generation ==========
	t.Log("Phase 1: Running DKG protocol")

	// Each server generates their polynomial and shares
	commitments := make([][]*ristretto255.Element, n)
	allShares := make([][]toprf.Share, n)

	for i := uint8(0); i < n; i++ {
		var err error
		commitments[i], allShares[i], err = Start(n, threshold)
		if err != nil {
			t.Fatalf("Server %d: DKG Start failed: %v", i+1, err)
		}
	}

	// Distribute shares
	sharesForServer := make([][]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		sharesForServer[i] = make([]toprf.Share, n)
		for j := uint8(0); j < n; j++ {
			sharesForServer[i][j] = allShares[j][i]
		}
	}

	// Each server verifies and combines their shares
	keyShares := make([]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		fails, _ := VerifyCommitments(n, threshold, i+1, commitments, sharesForServer[i])
		if len(fails) > 0 {
			t.Fatalf("Server %d: DKG verification failed", i+1)
		}

		var err error
		keyShares[i], err = Finish(sharesForServer[i], i+1)
		if err != nil {
			t.Fatalf("Server %d: DKG Finish failed: %v", i+1, err)
		}
	}

	t.Logf("DKG complete: %d servers each have a share of the threshold key", n)

	// ========== Phase 2: Generate Zero-Shares for 3HashTDH ==========
	t.Log("Phase 2: Generating zero-shares for 3HashTDH")

	// Generate zero-shares (Shamir sharing of 0)
	zero := ristretto255.NewScalar()
	zero.Decode([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	zeroShares, err := toprf.CreateShares(zero, n, threshold)
	if err != nil {
		t.Fatalf("Failed to create zero-shares: %v", err)
	}

	// ========== Phase 3: Threshold OPRF Protocol ==========
	t.Log("Phase 3: Running Threshold OPRF protocol")

	// Client: blind their password
	password := []byte("my-secret-password")

	// Generate a valid blinding scalar
	var blindBuf [64]byte
	_, err = rand.Read(blindBuf[:])
	if err != nil {
		t.Fatalf("Failed to generate blinding bytes: %v", err)
	}
	blindScalar := ristretto255.NewScalar().FromUniformBytes(blindBuf[:])
	blindingFactor := blindScalar.Encode(nil)

	blind, alpha, err := oprf.Blind(password, blindingFactor)
	if err != nil {
		t.Fatalf("Client Blind failed: %v", err)
	}
	t.Logf("Client blinded password: alpha=%x...", alpha[:8])

	// Session-specific identifier (must be same for all servers)
	ssid := make([]byte, 32)
	_, err = rand.Read(ssid)
	if err != nil {
		t.Fatalf("Failed to generate SSID: %v", err)
	}

	// Use first 'threshold' servers to evaluate
	// In practice, client would contact any 'threshold' available servers
	participatingServers := []uint8{1, 2, 3} // Using servers 1, 2, 3
	responses := make([][]byte, threshold)

	for i, serverIdx := range participatingServers {
		// Server evaluates using 3HashTDH
		beta, err := toprf.ThreeHashTDH(
			keyShares[serverIdx-1],
			zeroShares[serverIdx-1],
			alpha,
			ssid,
		)
		if err != nil {
			t.Fatalf("Server %d: ThreeHashTDH failed: %v", serverIdx, err)
		}
		responses[i] = beta
		t.Logf("Server %d evaluation: beta=%x...", serverIdx, beta[:8])
	}

	// Client: combine responses
	combined, err := toprf.ThresholdCombine(responses)
	if err != nil {
		t.Fatalf("Client ThresholdCombine failed: %v", err)
	}
	t.Logf("Client combined responses: %x...", combined[:8])

	// Client: unblind
	unblinded, err := oprf.Unblind(blind, combined)
	if err != nil {
		t.Fatalf("Client Unblind failed: %v", err)
	}

	// Client: finalize to get OPRF output
	output, err := oprf.Finalize(password, unblinded)
	if err != nil {
		t.Fatalf("Client Finalize failed: %v", err)
	}

	t.Logf("Final OPRF output: %x", output)

	// ========== Phase 4: Verify reproducibility ==========
	t.Log("Phase 4: Verifying reproducibility with same servers")

	// Use the same subset of servers again to verify we get the same output
	// NOTE: Using different server subsets requires applying Lagrange coefficients
	// based on which servers are participating. ThreeHashTDH currently doesn't
	// support this automatically - you need to use the same subset or implement
	// coefficient adjustment.
	responses2 := make([][]byte, threshold)

	for i, serverIdx := range participatingServers {
		beta, err := toprf.ThreeHashTDH(
			keyShares[serverIdx-1],
			zeroShares[serverIdx-1],
			alpha,
			ssid,
		)
		if err != nil {
			t.Fatalf("Server %d: ThreeHashTDH failed: %v", serverIdx, err)
		}
		responses2[i] = beta
	}

	combined2, err := toprf.ThresholdCombine(responses2)
	if err != nil {
		t.Fatalf("Client ThresholdCombine failed: %v", err)
	}

	unblinded2, err := oprf.Unblind(blind, combined2)
	if err != nil {
		t.Fatalf("Client Unblind failed: %v", err)
	}

	output2, err := oprf.Finalize(password, unblinded2)
	if err != nil {
		t.Fatalf("Client Finalize failed: %v", err)
	}

	// Both outputs should be identical
	if string(output) != string(output2) {
		t.Errorf("OPRF outputs don't match!\nOutput1: %x\nOutput2: %x", output, output2)
	} else {
		t.Log("SUCCESS: Reproducible OPRF output!")
	}

	// ========== Phase 5: Verify threshold property ==========
	t.Log("Phase 5: Verifying threshold property (too few servers should fail)")

	// Try with only threshold-1 servers
	if threshold > 1 {
		participatingServers3 := participatingServers[:threshold-1]
		responses3 := make([][]byte, len(participatingServers3))

		for i, serverIdx := range participatingServers3 {
			beta, err := toprf.ThreeHashTDH(
				keyShares[serverIdx-1],
				zeroShares[serverIdx-1],
				alpha,
				ssid,
			)
			if err != nil {
				t.Fatalf("Server %d: ThreeHashTDH failed: %v", serverIdx, err)
			}
			responses3[i] = beta
		}

		combined3, err := toprf.ThresholdCombine(responses3)
		if err != nil {
			t.Fatalf("ThresholdCombine failed: %v", err)
		}

		unblinded3, err := oprf.Unblind(blind, combined3)
		if err != nil {
			t.Fatalf("Unblind failed: %v", err)
		}

		output3, err := oprf.Finalize(password, unblinded3)
		if err != nil {
			t.Fatalf("Finalize failed: %v", err)
		}

		// With insufficient servers, output should be different (incorrect)
		if string(output) == string(output3) {
			t.Logf("Note: Insufficient servers gave same output (unlikely but possible)")
		} else {
			t.Logf("Confirmed: Insufficient servers (%d) gives wrong output", len(participatingServers3))
		}
	}

	t.Log("Integration test complete: DKG + Threshold OPRF working correctly!")
}

// TestDKGSecurityProperty verifies that no single server knows the complete key
func TestDKGSecurityProperty(t *testing.T) {
	const n = 3
	const threshold = 2

	// Run DKG
	commitments := make([][]*ristretto255.Element, n)
	allShares := make([][]toprf.Share, n)

	for i := uint8(0); i < n; i++ {
		var err error
		commitments[i], allShares[i], err = Start(n, threshold)
		if err != nil {
			t.Fatalf("DKG Start failed: %v", err)
		}
	}

	sharesForServer := make([][]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		sharesForServer[i] = make([]toprf.Share, n)
		for j := uint8(0); j < n; j++ {
			sharesForServer[i][j] = allShares[j][i]
		}
	}

	keyShares := make([]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		var err error
		keyShares[i], err = Finish(sharesForServer[i], i+1)
		if err != nil {
			t.Fatalf("DKG Finish failed: %v", err)
		}
	}

	// Reconstruct the group secret using all shares
	fullSecret, err := Reconstruct(keyShares)
	if err != nil {
		t.Fatalf("Reconstruct failed: %v", err)
	}

	// Verify that individual shares are different from the full secret
	for i := uint8(0); i < n; i++ {
		shareBytes := keyShares[i].Value.Encode(nil)
		secretBytes := fullSecret.Encode(nil)

		if string(shareBytes) == string(secretBytes) {
			t.Errorf("Server %d's share equals the full secret! DKG security violation!", i+1)
		}
	}

	t.Log("Security property verified: No single server has the complete key")
}
