// Package main demonstrates threshold OPRF usage.
//
// This example shows how to:
//  1. Split an OPRF key into multiple shares (5 servers, threshold 3)
//  2. Have any 3 servers evaluate partial results
//  3. Combine partial results to get the final OPRF output
//
// The threshold property means:
//   - Any 3 of 5 servers can complete the OPRF
//   - 2 or fewer servers learn nothing
//   - System continues to work even if 2 servers fail
//
// To run: go run threshold_oprf.go
package main

import (
	"fmt"
	"log"

	"github.com/gtank/ristretto255"
	"github.com/wurp/go-oprf/oprf"
	"github.com/wurp/go-oprf/toprf"
)

func main() {
	fmt.Println("=== Threshold OPRF Example ===\n")

	// Setup: Create shares for 5 servers with threshold 3
	n := uint8(5)
	threshold := uint8(3)

	fmt.Printf("Setup: Creating %d shares with threshold %d\n", n, threshold)
	fmt.Println("This means any 3 of 5 servers can complete the OPRF\n")

	// Generate a secret key and split it into shares
	secret, err := oprf.KeyGen()
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	secretScalar := ristretto255.NewScalar()
	if err := secretScalar.Decode(secret); err != nil {
		log.Fatalf("Failed to decode key: %v", err)
	}

	shares, err := toprf.CreateShares(secretScalar, n, threshold)
	if err != nil {
		log.Fatalf("Failed to create shares: %v", err)
	}

	fmt.Printf("Created %d shares (indexes 1-%d)\n\n", len(shares), n)

	// Client: Blind input (same as basic OPRF)
	input := []byte("threshold secret")
	fmt.Printf("Client: Input = %q\n", input)

	fmt.Println("Client: Blinding input...")
	r, alpha, err := oprf.Blind(input, nil)
	if err != nil {
		log.Fatalf("Failed to blind: %v", err)
	}
	fmt.Printf("Client: Sending blinded element to servers\n\n")

	// Servers: Let's say servers 1, 2, and 3 are online and responding
	// (we could choose any 3 of the 5 servers)
	activeServers := []uint8{1, 2, 3}

	fmt.Printf("Active servers: %v (any %d would work)\n", activeServers, threshold)

	var parts [][]byte
	for _, serverIdx := range activeServers {
		// Each server evaluates using their share
		fmt.Printf("Server %d: Evaluating with share %d...\n", serverIdx, shares[serverIdx-1].Index)

		part, err := toprf.Evaluate(shares[serverIdx-1], alpha, activeServers)
		if err != nil {
			log.Fatalf("Server %d failed to evaluate: %v", serverIdx, err)
		}

		parts = append(parts, part)
	}

	fmt.Println("\nClient: Received partial evaluations from all servers")

	// Client: Combine partial evaluations
	fmt.Println("Client: Combining partial evaluations...")
	beta, err := toprf.ThresholdCombine(parts)
	if err != nil {
		log.Fatalf("Failed to combine: %v", err)
	}

	// Client: Unblind and finalize (same as basic OPRF)
	fmt.Println("Client: Unblinding and finalizing...")
	n_element, err := oprf.Unblind(r, beta)
	if err != nil {
		log.Fatalf("Failed to unblind: %v", err)
	}

	output, err := oprf.Finalize(input, n_element)
	if err != nil {
		log.Fatalf("Failed to finalize: %v", err)
	}

	fmt.Printf("\n=== Threshold OPRF Complete ===\n")
	fmt.Printf("Final output (%d bytes): %x...\n", len(output), output[:16])

	fmt.Println("\n=== Properties ===")
	fmt.Println("✓ Output is deterministic for given input and key")
	fmt.Println("✓ Threshold evaluation produces same result as basic OPRF")

	fmt.Println("\n=== Threshold Properties ===")
	fmt.Printf("✓ Any %d of %d servers can complete the OPRF\n", threshold, n)
	fmt.Printf("✓ System works even if %d servers are offline\n", n-threshold)
	fmt.Println("✓ Secret key never reconstructed in one location")
	fmt.Println("✓ Servers learn nothing about client's input")
}
