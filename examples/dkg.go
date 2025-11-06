// Package main demonstrates Distributed Key Generation (DKG) usage.
//
// This example shows how multiple participants can collaboratively generate
// a shared secret key without any single participant knowing the complete key.
//
// The protocol has three phases:
//  1. Start: Each participant generates polynomial and shares
//  2. Verify: Participants verify received shares against commitments
//  3. Finish: Participants combine shares to get final secret share
//
// To run: go run dkg.go
package main

import (
	"fmt"
	"log"

	"github.com/gtank/ristretto255"
	"github.com/wurp/go-oprf/dkg"
	"github.com/wurp/go-oprf/toprf"
)

func main() {
	fmt.Println("=== Distributed Key Generation Example ===\n")

	// Setup: 3 participants with threshold 2
	n := uint8(3)
	threshold := uint8(2)

	fmt.Printf("Protocol: %d participants, threshold %d\n", n, threshold)
	fmt.Println("Any 2 participants can use the key, but 1 learns nothing\n")

	// ==================== Phase 1: Start ====================
	fmt.Println("=== Phase 1: Start ===")
	fmt.Println("Each participant generates polynomial and creates shares\n")

	// Each participant runs dkg.Start independently
	commitments1, shares1, err := dkg.Start(n, threshold)
	if err != nil {
		log.Fatalf("Participant 1 failed to start: %v", err)
	}
	fmt.Printf("Participant 1: Generated %d commitments and %d shares\n", len(commitments1), len(shares1))

	commitments2, shares2, err := dkg.Start(n, threshold)
	if err != nil {
		log.Fatalf("Participant 2 failed to start: %v", err)
	}
	fmt.Printf("Participant 2: Generated %d commitments and %d shares\n", len(commitments2), len(shares2))

	commitments3, shares3, err := dkg.Start(n, threshold)
	if err != nil {
		log.Fatalf("Participant 3 failed to start: %v", err)
	}
	fmt.Printf("Participant 3: Generated %d commitments and %d shares\n\n", len(commitments3), len(shares3))

	// In a real system:
	// - Commitments are broadcast to all participants (public channel)
	// - Shares are sent privately to each participant (secure channel)
	//   - shares1[0] goes to participant 1
	//   - shares1[1] goes to participant 2
	//   - shares1[2] goes to participant 3
	//   - etc.

	fmt.Println("Participants exchange:")
	fmt.Println("- Commitments (broadcast to everyone)")
	fmt.Println("- Shares (sent privately to each participant)\n")

	// ==================== Phase 2: Verify ====================
	fmt.Println("=== Phase 2: Verify ===")
	fmt.Println("Each participant verifies received shares match commitments\n")

	// Participant 1 verifies their received shares
	allCommitments1 := [][]*ristretto255.Element{commitments1, commitments2, commitments3}
	receivedShares1 := []toprf.Share{shares1[0], shares2[0], shares3[0]}

	fails1, err := dkg.VerifyCommitments(n, threshold, 1, allCommitments1, receivedShares1)
	if err != nil {
		log.Fatalf("Participant 1 verification error: %v", err)
	}
	if len(fails1) > 0 {
		log.Fatalf("Participant 1: Verification failed for participants: %v", fails1)
	}
	fmt.Println("Participant 1: ✓ All shares verified")

	// Participant 2 verifies their received shares
	allCommitments2 := [][]*ristretto255.Element{commitments1, commitments2, commitments3}
	receivedShares2 := []toprf.Share{shares1[1], shares2[1], shares3[1]}

	fails2, err := dkg.VerifyCommitments(n, threshold, 2, allCommitments2, receivedShares2)
	if err != nil {
		log.Fatalf("Participant 2 verification error: %v", err)
	}
	if len(fails2) > 0 {
		log.Fatalf("Participant 2: Verification failed for participants: %v", fails2)
	}
	fmt.Println("Participant 2: ✓ All shares verified")

	// Participant 3 verifies their received shares
	allCommitments3 := [][]*ristretto255.Element{commitments1, commitments2, commitments3}
	receivedShares3 := []toprf.Share{shares1[2], shares2[2], shares3[2]}

	fails3, err := dkg.VerifyCommitments(n, threshold, 3, allCommitments3, receivedShares3)
	if err != nil {
		log.Fatalf("Participant 3 verification error: %v", err)
	}
	if len(fails3) > 0 {
		log.Fatalf("Participant 3: Verification failed for participants: %v", fails3)
	}
	fmt.Println("Participant 3: ✓ All shares verified\n")

	// ==================== Phase 3: Finish ====================
	fmt.Println("=== Phase 3: Finish ===")
	fmt.Println("Each participant combines shares to get final secret share\n")

	// Each participant combines the shares they received
	finalShare1, err := dkg.Finish(receivedShares1, 1)
	if err != nil {
		log.Fatalf("Participant 1 failed to finish: %v", err)
	}
	fmt.Printf("Participant 1: Final share index=%d\n", finalShare1.Index)

	finalShare2, err := dkg.Finish(receivedShares2, 2)
	if err != nil {
		log.Fatalf("Participant 2 failed to finish: %v", err)
	}
	fmt.Printf("Participant 2: Final share index=%d\n", finalShare2.Index)

	finalShare3, err := dkg.Finish(receivedShares3, 3)
	if err != nil {
		log.Fatalf("Participant 3 failed to finish: %v", err)
	}
	fmt.Printf("Participant 3: Final share index=%d\n\n", finalShare3.Index)

	// ==================== Demonstration ====================
	fmt.Println("=== Demonstration: Threshold Reconstruction ===")
	fmt.Println("Any 2 participants can reconstruct the group secret\n")

	// Reconstruct using participants 1 and 2
	secret12, err := dkg.Reconstruct([]toprf.Share{finalShare1, finalShare2})
	if err != nil {
		log.Fatalf("Failed to reconstruct with shares 1,2: %v", err)
	}
	fmt.Printf("Secret from participants 1,2: %x...\n", secret12.Encode(nil)[:8])

	// Reconstruct using participants 1 and 3
	secret13, err := dkg.Reconstruct([]toprf.Share{finalShare1, finalShare3})
	if err != nil {
		log.Fatalf("Failed to reconstruct with shares 1,3: %v", err)
	}
	fmt.Printf("Secret from participants 1,3: %x...\n", secret13.Encode(nil)[:8])

	// Reconstruct using participants 2 and 3
	secret23, err := dkg.Reconstruct([]toprf.Share{finalShare2, finalShare3})
	if err != nil {
		log.Fatalf("Failed to reconstruct with shares 2,3: %v", err)
	}
	fmt.Printf("Secret from participants 2,3: %x...\n", secret23.Encode(nil)[:8])

	// Verify all reconstructions give the same secret
	secret12Bytes := secret12.Encode(nil)
	secret13Bytes := secret13.Encode(nil)
	secret23Bytes := secret23.Encode(nil)

	if string(secret12Bytes) != string(secret13Bytes) || string(secret12Bytes) != string(secret23Bytes) {
		log.Fatal("ERROR: Reconstructed secrets don't match!")
	}

	fmt.Println("\n✓ All combinations reconstruct the same secret")

	fmt.Println("\n=== DKG Properties ===")
	fmt.Println("✓ Group secret generated collaboratively")
	fmt.Println("✓ No participant knows the complete secret")
	fmt.Printf("✓ Any %d participants can reconstruct the secret\n", threshold)
	fmt.Println("✓ Fewer participants learn nothing")
	fmt.Println("✓ Commitments prevent malicious share dealing")
	fmt.Println("\n=== Use Case ===")
	fmt.Println("These shares can now be used for:")
	fmt.Println("- Threshold OPRF (see threshold_oprf.go)")
	fmt.Println("- Threshold signatures")
	fmt.Println("- Distributed decryption")
	fmt.Println("- Any threshold cryptography protocol")
}
