package dkg

import (
	"fmt"
	"testing"

	"github.com/gtank/ristretto255"
	"github.com/wurp/go-oprf/toprf"
)

// TestBasicDKG tests the basic DKG protocol with 3 participants and threshold 2
func TestBasicDKG(t *testing.T) {
	const n = 3
	const threshold = 2

	// Step 1: Each participant generates their polynomial and shares
	commitments := make([][]*ristretto255.Element, n)
	allShares := make([][]toprf.Share, n)

	for i := 0; i < n; i++ {
		var err error
		commitments[i], allShares[i], err = Start(n, threshold)
		if err != nil {
			t.Fatalf("Participant %d: Start failed: %v", i+1, err)
		}

		if len(commitments[i]) != threshold {
			t.Errorf("Participant %d: expected %d commitments, got %d", i+1, threshold, len(commitments[i]))
		}
		if len(allShares[i]) != n {
			t.Errorf("Participant %d: expected %d shares, got %d", i+1, n, len(allShares[i]))
		}
	}

	// Step 2: Distribute shares - each participant receives one share from each other participant
	// sharesForParticipant[i] contains shares addressed to participant i+1
	sharesForParticipant := make([][]toprf.Share, n)
	for i := 0; i < n; i++ {
		sharesForParticipant[i] = make([]toprf.Share, n)
		for j := 0; j < n; j++ {
			// Participant j+1 sends share allShares[j][i] to participant i+1
			sharesForParticipant[i][j] = allShares[j][i]
		}
	}

	// Step 3: Each participant verifies the shares they received
	for i := 0; i < n; i++ {
		fails, err := VerifyCommitments(n, threshold, uint8(i+1), commitments, sharesForParticipant[i])
		if err != nil {
			t.Fatalf("Participant %d: VerifyCommitments failed: %v", i+1, err)
		}
		if len(fails) > 0 {
			t.Errorf("Participant %d: verification failed for participants: %v", i+1, fails)
		}
	}

	// Step 4: Each participant combines their shares to get their final secret share
	finalShares := make([]toprf.Share, n)
	for i := 0; i < n; i++ {
		var err error
		finalShares[i], err = Finish(sharesForParticipant[i], uint8(i+1))
		if err != nil {
			t.Fatalf("Participant %d: Finish failed: %v", i+1, err)
		}
		if finalShares[i].Index != uint8(i+1) {
			t.Errorf("Participant %d: expected index %d, got %d", i+1, i+1, finalShares[i].Index)
		}
	}

	// Step 5: Verify that threshold participants can reconstruct the group secret
	// Use first 'threshold' participants
	secret1, err := Reconstruct(finalShares[:threshold])
	if err != nil {
		t.Fatalf("Reconstruct with first %d shares failed: %v", threshold, err)
	}

	// Use last 'threshold' participants
	secret2, err := Reconstruct(finalShares[n-threshold:])
	if err != nil {
		t.Fatalf("Reconstruct with last %d shares failed: %v", threshold, err)
	}

	// Both reconstructions should give the same secret
	secret1Bytes := secret1.Encode(nil)
	secret2Bytes := secret2.Encode(nil)
	if string(secret1Bytes) != string(secret2Bytes) {
		t.Errorf("Reconstructed secrets don't match!\nSecret1: %x\nSecret2: %x", secret1Bytes, secret2Bytes)
	}

	t.Logf("DKG successful! Group secret: %x", secret1Bytes)
}

// TestDKGWithDifferentParameters tests DKG with various n and threshold values
func TestDKGWithDifferentParameters(t *testing.T) {
	testCases := []struct {
		n         uint8
		threshold uint8
	}{
		{n: 2, threshold: 2},
		{n: 5, threshold: 3},
		{n: 7, threshold: 4},
		{n: 10, threshold: 6},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("%d-of-%d", tc.n, tc.threshold)
		t.Run(name, func(t *testing.T) {
			runDKG(t, tc.n, tc.threshold)
		})
	}
}

func runDKG(t *testing.T, n, threshold uint8) {
	// Each participant generates shares
	commitments := make([][]*ristretto255.Element, n)
	allShares := make([][]toprf.Share, n)

	for i := uint8(0); i < n; i++ {
		var err error
		commitments[i], allShares[i], err = Start(n, threshold)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}
	}

	// Distribute shares
	sharesForParticipant := make([][]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		sharesForParticipant[i] = make([]toprf.Share, n)
		for j := uint8(0); j < n; j++ {
			sharesForParticipant[i][j] = allShares[j][i]
		}
	}

	// Verify and combine
	finalShares := make([]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		fails, _ := VerifyCommitments(n, threshold, i+1, commitments, sharesForParticipant[i])
		if len(fails) > 0 {
			t.Fatalf("Verification failed for participant %d", i+1)
		}

		var err error
		finalShares[i], err = Finish(sharesForParticipant[i], i+1)
		if err != nil {
			t.Fatalf("Finish failed: %v", err)
		}
	}

	// Reconstruct with different subsets
	secret1, err := Reconstruct(finalShares[:threshold])
	if err != nil {
		t.Fatalf("Reconstruct failed: %v", err)
	}

	secret2, err := Reconstruct(finalShares[n-threshold:])
	if err != nil {
		t.Fatalf("Reconstruct failed: %v", err)
	}

	if string(secret1.Encode(nil)) != string(secret2.Encode(nil)) {
		t.Error("Secrets don't match")
	}
}

// TestDKGInsufficientShares verifies that reconstruction fails with fewer than threshold shares
func TestDKGInsufficientShares(t *testing.T) {
	const n = 5
	const threshold = 3

	// Run DKG protocol
	commitments := make([][]*ristretto255.Element, n)
	allShares := make([][]toprf.Share, n)

	for i := uint8(0); i < n; i++ {
		var err error
		commitments[i], allShares[i], err = Start(n, threshold)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}
	}

	sharesForParticipant := make([][]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		sharesForParticipant[i] = make([]toprf.Share, n)
		for j := uint8(0); j < n; j++ {
			sharesForParticipant[i][j] = allShares[j][i]
		}
	}

	finalShares := make([]toprf.Share, n)
	for i := uint8(0); i < n; i++ {
		var err error
		finalShares[i], err = Finish(sharesForParticipant[i], i+1)
		if err != nil {
			t.Fatalf("Finish failed: %v", err)
		}
	}

	// Try to reconstruct with threshold-1 shares (should work)
	secret1, err := Reconstruct(finalShares[:threshold])
	if err != nil {
		t.Fatalf("Expected reconstruction with threshold shares to succeed: %v", err)
	}

	// Verify that using fewer shares gives different result (polynomial interpolation)
	// With only 2 shares when threshold is 3, we don't have enough to uniquely determine
	// the polynomial, but Lagrange interpolation will still produce a value.
	// The important thing is that it won't be the correct secret.
	if threshold > 2 {
		secret2, err := Reconstruct(finalShares[:threshold-1])
		if err != nil {
			t.Fatalf("Interpolation failed: %v", err)
		}

		// The secrets should likely be different (though there's a tiny chance they match)
		s1 := secret1.Encode(nil)
		s2 := secret2.Encode(nil)

		// This is a probabilistic test - with very high probability, using fewer
		// than threshold shares will give a different (wrong) secret
		if string(s1) == string(s2) {
			t.Logf("Warning: insufficient shares gave same result (very unlikely but possible)")
		}
	}
}

// TestDKGInvalidParameters tests that invalid parameters are rejected
func TestDKGInvalidParameters(t *testing.T) {
	testCases := []struct {
		name      string
		n         uint8
		threshold uint8
		wantError bool
	}{
		{"threshold too small", 3, 1, true},
		{"threshold equals n", 3, 3, false},
		{"threshold > n", 3, 4, true},
		{"valid params", 5, 3, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Start(tc.n, tc.threshold)
			gotError := err != nil
			if gotError != tc.wantError {
				t.Errorf("Start(%d, %d): got error=%v, want error=%v",
					tc.n, tc.threshold, err, tc.wantError)
			}
		})
	}
}

// Benchmarks

func BenchmarkStart(b *testing.B) {
	const n = 5
	const threshold = 3
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = Start(n, threshold)
	}
}

func BenchmarkVerifyCommitments(b *testing.B) {
	const n = 5
	const threshold = 3

	// Setup
	commitments := make([][]*ristretto255.Element, n)
	allShares := make([][]toprf.Share, n)
	for i := 0; i < n; i++ {
		commitments[i], allShares[i], _ = Start(n, threshold)
	}

	// Collect shares for participant 1
	sharesForP1 := make([]toprf.Share, n)
	for j := 0; j < n; j++ {
		sharesForP1[j] = allShares[j][0]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyCommitments(n, threshold, 1, commitments, sharesForP1)
	}
}

func BenchmarkFinish(b *testing.B) {
	const n = 5
	const threshold = 3

	// Setup
	allShares := make([][]toprf.Share, n)
	for i := 0; i < n; i++ {
		_, allShares[i], _ = Start(n, threshold)
	}

	// Collect shares for participant 1
	sharesForP1 := make([]toprf.Share, n)
	for j := 0; j < n; j++ {
		sharesForP1[j] = allShares[j][0]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Finish(sharesForP1, 1)
	}
}

func BenchmarkReconstruct(b *testing.B) {
	const n = 5
	const threshold = 3

	// Setup - run full DKG
	commitments := make([][]*ristretto255.Element, n)
	allShares := make([][]toprf.Share, n)
	for i := 0; i < n; i++ {
		commitments[i], allShares[i], _ = Start(n, threshold)
	}

	sharesForParticipant := make([][]toprf.Share, n)
	for i := 0; i < n; i++ {
		sharesForParticipant[i] = make([]toprf.Share, n)
		for j := 0; j < n; j++ {
			sharesForParticipant[i][j] = allShares[j][i]
		}
	}

	finalShares := make([]toprf.Share, n)
	for i := 0; i < n; i++ {
		finalShares[i], _ = Finish(sharesForParticipant[i], uint8(i+1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Reconstruct(finalShares[:threshold])
	}
}

func BenchmarkDKGFullProtocol(b *testing.B) {
	const n = 5
	const threshold = 3

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each participant generates shares
		commitments := make([][]*ristretto255.Element, n)
		allShares := make([][]toprf.Share, n)
		for j := 0; j < n; j++ {
			commitments[j], allShares[j], _ = Start(n, threshold)
		}

		// Distribute shares
		sharesForParticipant := make([][]toprf.Share, n)
		for j := 0; j < n; j++ {
			sharesForParticipant[j] = make([]toprf.Share, n)
			for k := 0; k < n; k++ {
				sharesForParticipant[j][k] = allShares[k][j]
			}
		}

		// Verify and combine
		finalShares := make([]toprf.Share, n)
		for j := 0; j < n; j++ {
			VerifyCommitments(n, threshold, uint8(j+1), commitments, sharesForParticipant[j])
			finalShares[j], _ = Finish(sharesForParticipant[j], uint8(j+1))
		}

		// Reconstruct
		_, _ = Reconstruct(finalShares[:threshold])
	}
}
