package oprf

import (
	"encoding/hex"
	"testing"

	"github.com/gtank/ristretto255"
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
	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			input := mustDecodeHex(tv.input)
			blind := mustDecodeHex(tv.blind)

			r, alpha, err := Blind(input, blind)
			if err != nil {
				t.Fatalf("Blind failed: %v", err)
			}

			// Verify r matches the provided blind
			if hex.EncodeToString(r) != tv.blind {
				t.Errorf("Returned blind mismatch:\ngot:  %s\nwant: %s", hex.EncodeToString(r), tv.blind)
			}

			// Verify blinded element matches expected value
			if hex.EncodeToString(alpha) != tv.blindedElement {
				t.Errorf("Blinded element mismatch:\ngot:  %s\nwant: %s", hex.EncodeToString(alpha), tv.blindedElement)
			}
		})
	}
}

// TestEvaluate tests the Evaluate function with test vectors
func TestEvaluate(t *testing.T) {
	privateKey := mustDecodeHex(testPrivateKey)

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			blindedElement := mustDecodeHex(tv.blindedElement)

			beta, err := Evaluate(privateKey, blindedElement)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			// Verify evaluation element matches expected value
			if hex.EncodeToString(beta) != tv.evaluationElement {
				t.Errorf("Evaluation element mismatch:\ngot:  %s\nwant: %s", hex.EncodeToString(beta), tv.evaluationElement)
			}
		})
	}
}

// TestUnblind tests the Unblind function with test vectors
func TestUnblind(t *testing.T) {
	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			blind := mustDecodeHex(tv.blind)
			evaluationElement := mustDecodeHex(tv.evaluationElement)

			n, err := Unblind(blind, evaluationElement)
			if err != nil {
				t.Fatalf("Unblind failed: %v", err)
			}

			// Verify n is a valid ristretto255 element (32 bytes)
			if len(n) != ElementBytes {
				t.Errorf("Unblinded element has wrong length: got %d, want %d", len(n), ElementBytes)
			}

			// Note: We don't have the expected N value in test vectors
			// It will be verified implicitly through Finalize in the end-to-end test
		})
	}
}

// TestFinalize tests the Finalize function with test vectors
func TestFinalize(t *testing.T) {
	privateKey := mustDecodeHex(testPrivateKey)

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			input := mustDecodeHex(tv.input)
			blind := mustDecodeHex(tv.blind)

			// First, we need to compute N through the full flow
			// Blind
			_, alpha, err := Blind(input, blind)
			if err != nil {
				t.Fatalf("Blind failed: %v", err)
			}

			// Evaluate
			beta, err := Evaluate(privateKey, alpha)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			// Unblind to get N
			n, err := Unblind(blind, beta)
			if err != nil {
				t.Fatalf("Unblind failed: %v", err)
			}

			// Now test Finalize
			output, err := Finalize(input, n)
			if err != nil {
				t.Fatalf("Finalize failed: %v", err)
			}

			// Verify output matches expected value
			if hex.EncodeToString(output) != tv.output {
				t.Errorf("Output mismatch:\ngot:  %s\nwant: %s", hex.EncodeToString(output), tv.output)
			}
		})
	}
}

// TestOPRFEndToEnd tests the complete OPRF protocol flow
func TestOPRFEndToEnd(t *testing.T) {
	privateKey := mustDecodeHex(testPrivateKey)

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			input := mustDecodeHex(tv.input)
			blind := mustDecodeHex(tv.blind)

			// Client: Blind
			r, alpha, err := Blind(input, blind)
			if err != nil {
				t.Fatalf("Blind failed: %v", err)
			}

			// Verify alpha matches expected
			if hex.EncodeToString(alpha) != tv.blindedElement {
				t.Errorf("Alpha mismatch:\ngot:  %s\nwant: %s", hex.EncodeToString(alpha), tv.blindedElement)
			}

			// Server: Evaluate
			beta, err := Evaluate(privateKey, alpha)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			// Verify beta matches expected
			if hex.EncodeToString(beta) != tv.evaluationElement {
				t.Errorf("Beta mismatch:\ngot:  %s\nwant: %s", hex.EncodeToString(beta), tv.evaluationElement)
			}

			// Client: Unblind
			n, err := Unblind(r, beta)
			if err != nil {
				t.Fatalf("Unblind failed: %v", err)
			}

			// Client: Finalize
			output, err := Finalize(input, n)
			if err != nil {
				t.Fatalf("Finalize failed: %v", err)
			}

			// Verify final output matches expected value
			if hex.EncodeToString(output) != tv.output {
				t.Errorf("Final output mismatch:\ngot:  %s\nwant: %s", hex.EncodeToString(output), tv.output)
			}
		})
	}
}

// TestKeyGen tests the KeyGen function
func TestKeyGen(t *testing.T) {
	// Generate multiple keys and verify they're valid
	for i := 0; i < 10; i++ {
		key, err := KeyGen()
		if err != nil {
			t.Fatalf("KeyGen failed: %v", err)
		}

		// Verify key length
		if len(key) != ScalarBytes {
			t.Errorf("Key has wrong length: got %d, want %d", len(key), ScalarBytes)
		}

		// Verify key can be decoded as a valid scalar
		scalar := ristretto255.NewScalar()
		if err := scalar.Decode(key); err != nil {
			t.Errorf("Generated key is not a valid scalar: %v", err)
		}
	}

	// Verify keys are different (not stuck on a fixed value)
	key1, _ := KeyGen()
	key2, _ := KeyGen()
	if hex.EncodeToString(key1) == hex.EncodeToString(key2) {
		t.Error("KeyGen generated identical keys (unlikely to be random)")
	}
}

// Benchmarks

func BenchmarkBlind(b *testing.B) {
	input := []byte("benchmark-password")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = Blind(input, nil)
	}
}

func BenchmarkEvaluate(b *testing.B) {
	privateKey := mustDecodeHex(testPrivateKey)
	input := []byte("benchmark-password")
	_, alpha, _ := Blind(input, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Evaluate(privateKey, alpha)
	}
}

func BenchmarkUnblind(b *testing.B) {
	privateKey := mustDecodeHex(testPrivateKey)
	input := []byte("benchmark-password")
	r, alpha, _ := Blind(input, nil)
	beta, _ := Evaluate(privateKey, alpha)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Unblind(r, beta)
	}
}

func BenchmarkFinalize(b *testing.B) {
	privateKey := mustDecodeHex(testPrivateKey)
	input := []byte("benchmark-password")
	r, alpha, _ := Blind(input, nil)
	beta, _ := Evaluate(privateKey, alpha)
	n, _ := Unblind(r, beta)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Finalize(input, n)
	}
}

func BenchmarkOPRFEndToEnd(b *testing.B) {
	privateKey := mustDecodeHex(testPrivateKey)
	input := []byte("benchmark-password")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, alpha, _ := Blind(input, nil)
		beta, _ := Evaluate(privateKey, alpha)
		n, _ := Unblind(r, beta)
		_, _ = Finalize(input, n)
	}
}

func BenchmarkKeyGen(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = KeyGen()
	}
}
