// Package main demonstrates basic OPRF usage.
//
// This example shows the complete OPRF protocol flow between a client and server:
//  1. Server generates a private key
//  2. Client blinds their input
//  3. Server evaluates the blinded input
//  4. Client unblinds and finalizes to get the OPRF output
//
// To run: go run basic_oprf.go
package main

import (
	"fmt"
	"log"

	"github.com/wurp/go-oprf/oprf"
)

func main() {
	fmt.Println("=== Basic OPRF Example ===\n")

	// Server: Generate a private key
	fmt.Println("Server: Generating private key...")
	privateKey, err := oprf.KeyGen()
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	fmt.Printf("Server: Generated %d-byte private key\n\n", len(privateKey))

	// Client: Prepare input and blind it
	input := []byte("my secret password")
	fmt.Printf("Client: Input = %q\n", input)

	fmt.Println("Client: Blinding input...")
	r, alpha, err := oprf.Blind(input, nil) // nil = generate random blind
	if err != nil {
		log.Fatalf("Failed to blind: %v", err)
	}
	fmt.Printf("Client: Generated blinded element (%d bytes)\n", len(alpha))
	fmt.Printf("Client: Blinding factor r = %x...\n\n", r[:8])

	// In a real system, the client would send alpha to the server over the network

	// Server: Evaluate the blinded input
	fmt.Println("Server: Evaluating blinded input...")
	beta, err := oprf.Evaluate(privateKey, alpha)
	if err != nil {
		log.Fatalf("Failed to evaluate: %v", err)
	}
	fmt.Printf("Server: Generated evaluated element (%d bytes)\n\n", len(beta))

	// In a real system, the server would send beta back to the client

	// Client: Unblind and finalize
	fmt.Println("Client: Unblinding response...")
	n, err := oprf.Unblind(r, beta)
	if err != nil {
		log.Fatalf("Failed to unblind: %v", err)
	}

	fmt.Println("Client: Finalizing OPRF output...")
	output, err := oprf.Finalize(input, n)
	if err != nil {
		log.Fatalf("Failed to finalize: %v", err)
	}

	fmt.Printf("\n=== OPRF Complete ===\n")
	fmt.Printf("Final output (%d bytes): %x...\n", len(output), output[:16])

	// The output is a deterministic pseudorandom value derived from:
	// - The server's private key (server controls)
	// - The client's input (client controls)
	// Neither party learns the other's input!

	fmt.Println("\n=== Privacy Properties ===")
	fmt.Println("✓ Server never learned the client's input")
	fmt.Println("✓ Client never learned the server's key")
	fmt.Println("✓ Output is deterministic: same input+key = same output")
}
