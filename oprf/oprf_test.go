package oprf

import (
	"encoding/hex"
	"testing"
)

// Test vectors from IRTF CFRG OPRF specification
// Source: https://github.com/cfrg/draft-irtf-cfrg-voprf
// Extracted from liboprf test suite: ~/projects/git/liboprf/src/tests/testvecs2h.py

const (
	// Server private key used for all test cases
	testPrivateKey = "5ebcea5ee37023ccb9fc2d2019f9d7737be85591ae8652ffa9ef0f4d37063b0e"
)

// Test vector structure
type oprfTestVector struct {
	name              string
	input             string
	blind             string
	blindedElement    string
	evaluationElement string
	output            string
}

var testVectors = []oprfTestVector{
	{
		name:              "single byte input",
		input:             "00",
		blind:             "64d37aed22a27f5191de1c1d69fadb899d8862b58eb4220029e036ec4c1f6706",
		blindedElement:    "609a0ae68c15a3cf6903766461307e5c8bb2f95e7e6550e1ffa2dc99e412803c",
		evaluationElement: "7ec6578ae5120958eb2db1745758ff379e77cb64fe77b0b2d8cc917ea0869c7e",
		output:            "527759c3d9366f277d8c6020418d96bb393ba2afb20ff90df23fb7708264e2f3ab9135e3bd69955851de4b1f9fe8a0973396719b7912ba9ee8aa7d0b5e24bcf6",
	},
	{
		name:              "repeated byte pattern",
		input:             "5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a",
		blind:             "64d37aed22a27f5191de1c1d69fadb899d8862b58eb4220029e036ec4c1f6706",
		blindedElement:    "da27ef466870f5f15296299850aa088629945a17d1f5b7f5ff043f76b3c06418",
		evaluationElement: "b4cbf5a4f1eeda5a63ce7b77c7d23f461db3fcab0dd28e4e17cecb5c90d02c25",
		output:            "f4a74c9c592497375e796aa837e907b1a045d34306a749db9f34221f7e750cb4f2a6413a6bf6fa5e19ba6348eb673934a722a7ede2e7621306d18951e7cf2c73",
	},
}

// Helper function to decode hex strings
func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in test vector: " + err.Error())
	}
	return b
}

// TestBlind tests the Blind function with test vectors
func TestBlind(t *testing.T) {
	t.Skip("TODO: implement oprf.Blind")

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			input := mustDecodeHex(tv.input)
			blind := mustDecodeHex(tv.blind)
			expectedBlinded := mustDecodeHex(tv.blindedElement)
			_, _, _ = input, blind, expectedBlinded

			// TODO: Implement Blind function
			// r, alpha, err := Blind(input, blind)
			// if err != nil {
			//     t.Fatalf("Blind failed: %v", err)
			// }

			// Verify blinded element matches expected value
			// if !bytes.Equal(alpha, expectedBlinded) {
			//     t.Errorf("Blinded element mismatch:\ngot:  %x\nwant: %x", alpha, expectedBlinded)
			// }
		})
	}
}

// TestEvaluate tests the Evaluate function with test vectors
func TestEvaluate(t *testing.T) {
	t.Skip("TODO: implement oprf.Evaluate")

	privateKey := mustDecodeHex(testPrivateKey)
	_ = privateKey

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			blindedElement := mustDecodeHex(tv.blindedElement)
			expectedEvaluation := mustDecodeHex(tv.evaluationElement)
			_, _ = blindedElement, expectedEvaluation

			// TODO: Implement Evaluate function
			// beta, err := Evaluate(privateKey, blindedElement)
			// if err != nil {
			//     t.Fatalf("Evaluate failed: %v", err)
			// }

			// Verify evaluation element matches expected value
			// if !bytes.Equal(beta, expectedEvaluation) {
			//     t.Errorf("Evaluation element mismatch:\ngot:  %x\nwant: %x", beta, expectedEvaluation)
			// }
		})
	}
}

// TestUnblind tests the Unblind function with test vectors
func TestUnblind(t *testing.T) {
	t.Skip("TODO: implement oprf.Unblind")

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			blind := mustDecodeHex(tv.blind)
			evaluationElement := mustDecodeHex(tv.evaluationElement)
			_, _ = blind, evaluationElement

			// TODO: Implement Unblind function
			// n, err := Unblind(blind, evaluationElement)
			// if err != nil {
			//     t.Fatalf("Unblind failed: %v", err)
			// }

			// Note: We don't have the expected N value in test vectors
			// It will be verified implicitly through Finalize
		})
	}
}

// TestFinalize tests the Finalize function with test vectors
func TestFinalize(t *testing.T) {
	t.Skip("TODO: implement oprf.Finalize")

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			input := mustDecodeHex(tv.input)
			expectedOutput := mustDecodeHex(tv.output)
			_, _ = input, expectedOutput

			// TODO: Need to get N from Unblind first
			// For now, this test is a placeholder

			// output, err := Finalize(input, n)
			// if err != nil {
			//     t.Fatalf("Finalize failed: %v", err)
			// }

			// Verify output matches expected value
			// if !bytes.Equal(output, expectedOutput) {
			//     t.Errorf("Output mismatch:\ngot:  %x\nwant: %x", output, expectedOutput)
			// }
		})
	}
}

// TestOPRFEndToEnd tests the complete OPRF protocol flow
func TestOPRFEndToEnd(t *testing.T) {
	t.Skip("TODO: implement complete OPRF flow")

	privateKey := mustDecodeHex(testPrivateKey)
	_ = privateKey

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			input := mustDecodeHex(tv.input)
			blind := mustDecodeHex(tv.blind)
			expectedOutput := mustDecodeHex(tv.output)
			_, _, _ = input, blind, expectedOutput

			// Client: Blind
			// r, alpha, err := Blind(input, blind)
			// if err != nil {
			//     t.Fatalf("Blind failed: %v", err)
			// }

			// Server: Evaluate
			// beta, err := Evaluate(privateKey, alpha)
			// if err != nil {
			//     t.Fatalf("Evaluate failed: %v", err)
			// }

			// Client: Unblind
			// n, err := Unblind(r, beta)
			// if err != nil {
			//     t.Fatalf("Unblind failed: %v", err)
			// }

			// Client: Finalize
			// output, err := Finalize(input, n)
			// if err != nil {
			//     t.Fatalf("Finalize failed: %v", err)
			// }

			// Verify final output matches expected value
			// if !bytes.Equal(output, expectedOutput) {
			//     t.Errorf("Final output mismatch:\ngot:  %x\nwant: %x", output, expectedOutput)
			// }
		})
	}
}

// TestExpandMessageXMD tests the expand_message_xmd function
func TestExpandMessageXMD(t *testing.T) {
	t.Skip("TODO: implement expand_message_xmd")

	// TODO: Add test vectors for expand_message_xmd
	// This is a critical function for hash-to-curve
}

// TestHashToGroup tests the hash-to-group function
func TestHashToGroup(t *testing.T) {
	t.Skip("TODO: implement hash_to_group")

	// TODO: Add test vectors for hash-to-group
	// This function maps arbitrary input to ristretto255 points
}
