package toprf

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/gtank/ristretto255"
	"github.com/wurp/go-oprf/oprf"
)

// TestScalarFromUint8 tests conversion from uint8 to scalar
func TestScalarFromUint8(t *testing.T) {
	tests := []struct {
		val      uint8
		expected string // hex encoding of expected scalar
	}{
		{0, "0000000000000000000000000000000000000000000000000000000000000000"},
		{1, "0100000000000000000000000000000000000000000000000000000000000000"},
		{255, "ff00000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, tt := range tests {
		s := scalarFromUint8(tt.val)
		encoded := s.Encode(nil)
		got := hex.EncodeToString(encoded)
		if got != tt.expected {
			t.Errorf("scalarFromUint8(%d) = %s, want %s", tt.val, got, tt.expected)
		}
	}
}

// TestLagrangeCoefficients tests Lagrange coefficient computation
func TestLagrangeCoefficients(t *testing.T) {
	// Test with simple case: peers = [1, 2, 3], compute coeff for peer 1
	// For f(0): L_1(0) = (0-2)(0-3) / (1-2)(1-3) = 6/2 = 3
	peers := []uint8{1, 2, 3}
	c := coeff(1, peers)

	// Expected: 3 as scalar
	expected := scalarFromUint8(3)
	if !bytes.Equal(c.Encode(nil), expected.Encode(nil)) {
		t.Errorf("coeff(1, [1,2,3]) != 3")
	}
}

// TestCreateShares tests Shamir secret sharing
func TestCreateShares(t *testing.T) {
	// Create a secret
	secret := ristretto255.NewScalar()
	secretBytes, _ := hex.DecodeString("5ebcea5ee37023ccb9fc2d2019f9d7737be85591ae8652ffa9ef0f4d37063b0e")
	secret.Decode(secretBytes)

	// Create shares (n=5, threshold=3)
	shares, err := CreateShares(secret, 5, 3)
	if err != nil {
		t.Fatalf("CreateShares failed: %v", err)
	}

	if len(shares) != 5 {
		t.Errorf("Expected 5 shares, got %d", len(shares))
	}

	// Verify shares have correct indexes (1-5)
	for i, share := range shares {
		if share.Index != uint8(i+1) {
			t.Errorf("Share %d has wrong index: got %d, want %d", i, share.Index, i+1)
		}
	}

	// Test reconstruction with threshold shares (shares 1, 2, 3)
	thresholdShares := shares[0:3]
	reconstructed := interpolate(0, thresholdShares)

	if !bytes.Equal(reconstructed.Encode(nil), secret.Encode(nil)) {
		t.Errorf("Failed to reconstruct secret from threshold shares")
		t.Logf("Expected: %x", secret.Encode(nil))
		t.Logf("Got:      %x", reconstructed.Encode(nil))
	}

	// Test that threshold-1 shares cannot reconstruct (they can compute, but result is wrong)
	insufficientShares := shares[0:2]
	wrongReconstruction := interpolate(0, insufficientShares)

	if bytes.Equal(wrongReconstruction.Encode(nil), secret.Encode(nil)) {
		t.Errorf("Incorrectly reconstructed secret with insufficient shares")
	}
}

// TestThresholdOPRF tests the complete threshold OPRF flow
func TestThresholdOPRF(t *testing.T) {
	// 1. Generate a secret key
	keyBytes, err := oprf.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen failed: %v", err)
	}

	secret := ristretto255.NewScalar()
	secret.Decode(keyBytes)

	// 2. Split key into shares (n=3, threshold=2)
	shares, err := CreateShares(secret, 3, 2)
	if err != nil {
		t.Fatalf("CreateShares failed: %v", err)
	}

	// 3. Client blinds input
	input := []byte("password")
	r, alpha, err := oprf.Blind(input, nil)
	if err != nil {
		t.Fatalf("Blind failed: %v", err)
	}

	// 4. Threshold evaluation by servers 1 and 3
	indexes := []uint8{1, 3}
	responses := make([][]byte, 2)

	// Server 1 evaluates
	responses[0], err = Evaluate(shares[0], alpha, indexes)
	if err != nil {
		t.Fatalf("Server 1 Evaluate failed: %v", err)
	}

	// Server 3 evaluates
	responses[1], err = Evaluate(shares[2], alpha, indexes)
	if err != nil {
		t.Fatalf("Server 3 Evaluate failed: %v", err)
	}

	// 5. Client combines responses
	beta, err := ThresholdCombine(responses)
	if err != nil {
		t.Fatalf("ThresholdCombine failed: %v", err)
	}

	// 6. Client unblinds
	n, err := oprf.Unblind(r, beta)
	if err != nil {
		t.Fatalf("Unblind failed: %v", err)
	}

	// 7. Client finalizes
	output, err := oprf.Finalize(input, n)
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	// 8. Verify: compute non-threshold OPRF and compare
	betaNonThreshold, err := oprf.Evaluate(keyBytes, alpha)
	if err != nil {
		t.Fatalf("Non-threshold Evaluate failed: %v", err)
	}

	nNonThreshold, err := oprf.Unblind(r, betaNonThreshold)
	if err != nil {
		t.Fatalf("Non-threshold Unblind failed: %v", err)
	}

	outputNonThreshold, err := oprf.Finalize(input, nNonThreshold)
	if err != nil {
		t.Fatalf("Non-threshold Finalize failed: %v", err)
	}

	if !bytes.Equal(output, outputNonThreshold) {
		t.Errorf("Threshold and non-threshold outputs differ")
		t.Logf("Threshold output:     %x", output)
		t.Logf("Non-threshold output: %x", outputNonThreshold)
	}
}

// TestShareMarshal tests Share serialization/deserialization
func TestShareMarshal(t *testing.T) {
	// Create a share
	original := Share{
		Index: 42,
		Value: ristretto255.NewScalar(),
	}
	var randBytes [64]byte
	copy(randBytes[:], []byte("test data for scalar"))
	original.Value.FromUniformBytes(randBytes[:])

	// Marshal
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	if len(data) != ShareBytes {
		t.Errorf("Marshaled share has wrong length: got %d, want %d", len(data), ShareBytes)
	}

	// Unmarshal
	var decoded Share
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Verify
	if decoded.Index != original.Index {
		t.Errorf("Index mismatch: got %d, want %d", decoded.Index, original.Index)
	}

	if !bytes.Equal(decoded.Value.Encode(nil), original.Value.Encode(nil)) {
		t.Errorf("Value mismatch after marshal/unmarshal")
	}
}

// TestPartMarshal tests Part serialization/deserialization
func TestPartMarshal(t *testing.T) {
	// Create a part
	original := Part{
		Index:   7,
		Element: ristretto255.NewElement(),
	}
	// Set to non-identity element by multiplying generator by a scalar
	scalar := scalarFromUint8(123)
	original.Element.ScalarBaseMult(scalar)

	// Marshal
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	if len(data) != PartBytes {
		t.Errorf("Marshaled part has wrong length: got %d, want %d", len(data), PartBytes)
	}

	// Unmarshal
	var decoded Part
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Verify
	if decoded.Index != original.Index {
		t.Errorf("Index mismatch: got %d, want %d", decoded.Index, original.Index)
	}

	if !bytes.Equal(decoded.Element.Encode(nil), original.Element.Encode(nil)) {
		t.Errorf("Element mismatch after marshal/unmarshal")
	}
}

// TestThreeHashTDH tests the 3HashTDH protocol
func TestThreeHashTDH(t *testing.T) {
	// 1. Generate secret key and create shares
	keyBytes, err := oprf.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen failed: %v", err)
	}

	secret := ristretto255.NewScalar()
	secret.Decode(keyBytes)

	shares, err := CreateShares(secret, 3, 2)
	if err != nil {
		t.Fatalf("CreateShares failed: %v", err)
	}

	// 2. Create zero-sharing (polynomial that evaluates to 0)
	zero := ristretto255.NewScalar()
	zero.Decode([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	zeroShares, err := CreateShares(zero, 3, 2)
	if err != nil {
		t.Fatalf("CreateShares for zero failed: %v", err)
	}

	// 3. Client blinds input
	input := []byte("test-password")
	_, alpha, err := oprf.Blind(input, nil)
	if err != nil {
		t.Fatalf("Blind failed: %v", err)
	}

	// 4. Evaluate using 3HashTDH
	ssid := []byte("session-12345")

	response1, err := ThreeHashTDH(shares[0], zeroShares[0], alpha, ssid)
	if err != nil {
		t.Fatalf("ThreeHashTDH server 1 failed: %v", err)
	}

	response2, err := ThreeHashTDH(shares[1], zeroShares[1], alpha, ssid)
	if err != nil {
		t.Fatalf("ThreeHashTDH server 2 failed: %v", err)
	}

	// 5. Verify responses can be combined
	responses := [][]byte{response1, response2}
	_, err = ThresholdCombine(responses)
	if err != nil {
		t.Fatalf("ThresholdCombine for 3HashTDH failed: %v", err)
	}

	// Note: We can't easily verify correctness without full protocol,
	// but we can verify it doesn't crash and produces valid output
}

// TestInvalidInputs tests error handling
func TestInvalidInputs(t *testing.T) {
	// Test CreateShares with invalid parameters
	secret := ristretto255.NewScalar()

	_, err := CreateShares(secret, 2, 3) // threshold > n
	if err == nil {
		t.Error("CreateShares should fail when threshold > n")
	}

	_, err = CreateShares(secret, 5, 0) // threshold = 0
	if err == nil {
		t.Error("CreateShares should fail when threshold = 0")
	}

	// Test Evaluate with invalid blinded length
	share := Share{Index: 1, Value: secret}
	_, err = Evaluate(share, []byte{1, 2, 3}, []uint8{1, 2})
	if err == nil {
		t.Error("Evaluate should fail with invalid blinded length")
	}

	// Test ThresholdCombine with no responses
	_, err = ThresholdCombine([][]byte{})
	if err == nil {
		t.Error("ThresholdCombine should fail with no responses")
	}
}

// Benchmarks

func BenchmarkCreateShares(b *testing.B) {
	secret := ristretto255.NewScalar()
	secret.Decode([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CreateShares(secret, 5, 3)
	}
}

func BenchmarkEvaluate(b *testing.B) {
	secret := ristretto255.NewScalar()
	keyBytes, _ := oprf.KeyGen()
	secret.Decode(keyBytes)
	shares, _ := CreateShares(secret, 5, 3)
	input := []byte("benchmark-password")
	_, alpha, _ := oprf.Blind(input, nil)
	indexes := []uint8{1, 2, 3}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Evaluate(shares[0], alpha, indexes)
	}
}

func BenchmarkThresholdCombine(b *testing.B) {
	secret := ristretto255.NewScalar()
	keyBytes, _ := oprf.KeyGen()
	secret.Decode(keyBytes)
	shares, _ := CreateShares(secret, 5, 3)
	input := []byte("benchmark-password")
	_, alpha, _ := oprf.Blind(input, nil)
	indexes := []uint8{1, 2, 3}
	responses := make([][]byte, 3)
	responses[0], _ = Evaluate(shares[0], alpha, indexes)
	responses[1], _ = Evaluate(shares[1], alpha, indexes)
	responses[2], _ = Evaluate(shares[2], alpha, indexes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ThresholdCombine(responses)
	}
}

func BenchmarkThreeHashTDH(b *testing.B) {
	keyBytes, _ := oprf.KeyGen()
	secret := ristretto255.NewScalar()
	secret.Decode(keyBytes)
	shares, _ := CreateShares(secret, 3, 2)
	zero := ristretto255.NewScalar()
	zero.Decode([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	zeroShares, _ := CreateShares(zero, 3, 2)
	input := []byte("benchmark-password")
	_, alpha, _ := oprf.Blind(input, nil)
	ssid := []byte("session-id")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ThreeHashTDH(shares[0], zeroShares[0], alpha, ssid)
	}
}

func BenchmarkThresholdOPRFEndToEnd(b *testing.B) {
	keyBytes, _ := oprf.KeyGen()
	secret := ristretto255.NewScalar()
	secret.Decode(keyBytes)
	shares, _ := CreateShares(secret, 3, 2)
	input := []byte("benchmark-password")
	indexes := []uint8{1, 2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, alpha, _ := oprf.Blind(input, nil)
		resp1, _ := Evaluate(shares[0], alpha, indexes)
		resp2, _ := Evaluate(shares[1], alpha, indexes)
		beta, _ := ThresholdCombine([][]byte{resp1, resp2})
		n, _ := oprf.Unblind(r, beta)
		_, _ = oprf.Finalize(input, n)
	}
}
