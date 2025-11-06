// Package oprf implements the Oblivious Pseudorandom Function (OPRF) protocol
// using ristretto255 and SHA-512, following the IRTF CFRG specification.
//
// An OPRF is a two-party protocol between a client and server for computing
// a pseudorandom function (PRF) where the server holds the secret key and
// the client holds the input. The protocol ensures that:
//   - The server learns nothing about the client's input
//   - The client learns only the PRF output, not the server's key
//
// # Protocol Flow
//
// The basic OPRF protocol involves four steps:
//
//  1. Client blinds input using Blind():
//     Takes input and generates a random blinding factor r,
//     computes alpha = HashToGroup(input)^r
//
//  2. Server evaluates using Evaluate():
//     Computes beta = alpha^k where k is the server's private key
//
//  3. Client unblinds using Unblind():
//     Computes N = beta^(1/r) to remove the blinding factor
//
//  4. Client finalizes using Finalize():
//     Computes final output = Hash(input || N || "Finalize")
//
// # Usage Example
//
//	// Server: Generate a private key
//	privateKey, err := oprf.KeyGen()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Client: Blind the input
//	input := []byte("my secret input")
//	r, alpha, err := oprf.Blind(input, nil)  // nil = generate random blind
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send alpha to server...
//
//	// Server: Evaluate the blinded input
//	beta, err := oprf.Evaluate(privateKey, alpha)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send beta back to client...
//
//	// Client: Unblind and finalize
//	n, err := oprf.Unblind(r, beta)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	output, err := oprf.Finalize(input, n)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// output is the final OPRF result (64 bytes)
//
// # Cryptographic Details
//
// This implementation follows RFC 9497 (OPRF) and uses:
//   - Group: ristretto255 (RFC 9496)
//   - Hash: SHA-512
//   - Hash-to-curve: expand_message_xmd with SHA-512 (RFC 9380)
//
// All scalar operations are constant-time to prevent timing attacks.
//
// # Security Considerations
//
// - The blinding factor r must be randomly generated for each OPRF evaluation
// - The server's private key must be kept secret
// - This implementation provides computational privacy, not information-theoretic privacy
// - Side-channel protections rely on the underlying ristretto255 implementation
//
// # Compatibility
//
// This implementation is byte-for-byte compatible with liboprf (C implementation)
// and follows the IRTF CFRG OPRF specification test vectors.
package oprf

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gtank/ristretto255"
)

// Constants for OPRF implementation
const (
	// OPRF_BYTES is the output size of the OPRF (64 bytes for SHA-512)
	OPRF_BYTES = 64

	// ScalarBytes is the size of a ristretto255 scalar (32 bytes)
	ScalarBytes = 32

	// ElementBytes is the size of a ristretto255 element (32 bytes)
	ElementBytes = 32

	// HashBytes is the size used for hash-to-curve (64 bytes)
	HashBytes = 64
)

// Domain Separation Tags (DST) per RFC 9497
const (
	// HashToGroupDST is the domain separation tag for hash-to-group operations
	HashToGroupDST = "HashToGroup-OPRFV1-\x00-ristretto255-SHA512"

	// FinalizeDST is the domain separation tag for finalize operations
	FinalizeDST = "Finalize"
)

// SHA-512 parameters for expand_message_xmd
const (
	sha512OutputBytes = 64  // b_in_bytes: output size of SHA-512
	sha512BlockSize   = 128 // r_in_bytes: input block size of SHA-512
)

// expandMessageXMD implements expand_message_xmd from RFC 9380 Section 5.3.1
// using SHA-512 as the hash function.
//
// Parameters:
//   - msg: the message to expand
//   - dst: domain separation tag
//   - lenInBytes: desired output length in bytes
//
// Returns the expanded message as uniform bytes.
//
// This implements the expand_message_xmd algorithm:
// https://datatracker.ietf.org/doc/html/rfc9380#section-5.3.1
func expandMessageXMD(msg, dst []byte, lenInBytes int) ([]byte, error) {
	// ell = ceil(len_in_bytes / b_in_bytes)
	ell := (lenInBytes + sha512OutputBytes - 1) / sha512OutputBytes

	// Check ell is valid (must be <= 255)
	if ell > 255 {
		return nil, errors.New("lenInBytes too large for expand_message_xmd")
	}

	// DST_prime = DST || I2OSP(len(DST), 1)
	dstPrime := make([]byte, len(dst)+1)
	copy(dstPrime, dst)
	dstPrime[len(dst)] = byte(len(dst))

	// Z_pad = I2OSP(0, r_in_bytes) - block of zeros
	zPad := make([]byte, sha512BlockSize)

	// l_i_b_str = I2OSP(len_in_bytes, 2) - length as 2-byte big-endian
	libStr := make([]byte, 2)
	binary.BigEndian.PutUint16(libStr, uint16(lenInBytes))

	// msg_prime = Z_pad || msg || l_i_b_str || I2OSP(0, 1) || DST_prime
	h := sha512.New()
	h.Write(zPad)
	h.Write(msg)
	h.Write(libStr)
	h.Write([]byte{0})
	h.Write(dstPrime)

	// b_0 = H(msg_prime)
	b0 := h.Sum(nil)

	// b_1 = H(b_0 || I2OSP(1, 1) || DST_prime)
	h.Reset()
	h.Write(b0)
	h.Write([]byte{1})
	h.Write(dstPrime)
	b1 := h.Sum(nil)

	// uniformBytes will hold b_1 || b_2 || ... || b_ell
	uniformBytes := make([]byte, 0, ell*sha512OutputBytes)
	uniformBytes = append(uniformBytes, b1...)

	// Compute b_2, ..., b_ell
	bPrev := b1
	for i := 2; i <= ell; i++ {
		// b_i = H(strxor(b_0, b_(i-1)) || I2OSP(i, 1) || DST_prime)
		h.Reset()

		// Compute strxor(b_0, b_prev)
		xorResult := make([]byte, sha512OutputBytes)
		for j := 0; j < sha512OutputBytes; j++ {
			xorResult[j] = b0[j] ^ bPrev[j]
		}

		h.Write(xorResult)
		h.Write([]byte{byte(i)})
		h.Write(dstPrime)
		bi := h.Sum(nil)

		uniformBytes = append(uniformBytes, bi...)
		bPrev = bi
	}

	// Return the first len_in_bytes bytes
	return uniformBytes[:lenInBytes], nil
}

// hashToGroup implements the hash-to-group operation for ristretto255.
// It hashes arbitrary input to a ristretto255 curve point following RFC 9380.
//
// Parameters:
//   - msg: the message to hash to a group element
//
// Returns a ristretto255 element or an error.
//
// This follows the hash_to_ristretto255 specification using SHA-512.
func hashToGroup(msg []byte) (*ristretto255.Element, error) {
	// Expand message to 64 uniform bytes using expand_message_xmd
	// with the OPRF-specific domain separation tag
	uniformBytes, err := expandMessageXMD(msg, []byte(HashToGroupDST), HashBytes)
	if err != nil {
		return nil, fmt.Errorf("expand_message_xmd failed: %w", err)
	}

	// Map uniform bytes to ristretto255 element
	// FromUniformBytes maps 64 uniform bytes to a ristretto255 element
	element := ristretto255.NewElement()
	element.FromUniformBytes(uniformBytes)

	return element, nil
}

// Blind performs the client-side blinding operation in the OPRF protocol.
//
// Parameters:
//   - input: the input to be blinded
//   - blind: optional fixed blind value (for testing). If nil, a random blind is generated.
//
// Returns:
//   - r: the blinding scalar (32 bytes)
//   - alpha: the blinded element (32 bytes)
//   - error: any error that occurred
//
// The blind operation computes:
//  1. H0 = HashToGroup(input)
//  2. alpha = H0 * r (where r is the blinding scalar)
func Blind(input []byte, blind []byte) (r, alpha []byte, err error) {
	// Hash input to curve point
	h0, err := hashToGroup(input)
	if err != nil {
		return nil, nil, fmt.Errorf("hashToGroup failed: %w", err)
	}

	// Get or generate blinding scalar
	var rScalar *ristretto255.Scalar
	if blind != nil {
		// Use provided blind (for testing)
		if len(blind) != ScalarBytes {
			return nil, nil, fmt.Errorf("blind must be %d bytes, got %d", ScalarBytes, len(blind))
		}
		rScalar = ristretto255.NewScalar()
		if err := rScalar.Decode(blind); err != nil {
			return nil, nil, fmt.Errorf("invalid blind scalar: %w", err)
		}
		r = make([]byte, ScalarBytes)
		copy(r, blind)
	} else {
		// Generate random blinding scalar (for production use)
		// FromUniformBytes requires 64 random bytes
		randomBytes := make([]byte, 64)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, nil, fmt.Errorf("failed to generate random bytes: %w", err)
		}
		rScalar = ristretto255.NewScalar()
		rScalar.FromUniformBytes(randomBytes)
		r = rScalar.Encode(nil)
	}

	// Compute alpha = H0 * r (scalar multiplication)
	alphaElement := ristretto255.NewElement()
	alphaElement.ScalarMult(rScalar, h0)

	// Encode alpha as bytes
	alpha = alphaElement.Encode(nil)

	return r, alpha, nil
}

// Evaluate performs the server-side evaluation in the OPRF protocol.
//
// Parameters:
//   - k: the server's private key (32 bytes)
//   - alpha: the blinded element from the client (32 bytes)
//
// Returns:
//   - beta: the evaluated element (32 bytes)
//   - error: any error that occurred
//
// The evaluate operation computes:
//
//	beta = alpha^k (scalar multiplication)
func Evaluate(k []byte, alpha []byte) (beta []byte, err error) {
	// Validate private key length
	if len(k) != ScalarBytes {
		return nil, fmt.Errorf("private key must be %d bytes, got %d", ScalarBytes, len(k))
	}

	// Validate alpha length
	if len(alpha) != ElementBytes {
		return nil, fmt.Errorf("alpha must be %d bytes, got %d", ElementBytes, len(alpha))
	}

	// Decode private key as scalar
	kScalar := ristretto255.NewScalar()
	if err := kScalar.Decode(k); err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Decode alpha as element
	alphaElement := ristretto255.NewElement()
	if err := alphaElement.Decode(alpha); err != nil {
		return nil, fmt.Errorf("invalid alpha element: %w", err)
	}

	// Compute beta = alpha^k (scalar multiplication)
	betaElement := ristretto255.NewElement()
	betaElement.ScalarMult(kScalar, alphaElement)

	// Encode beta as bytes
	beta = betaElement.Encode(nil)

	return beta, nil
}

// Unblind performs the client-side unblinding operation in the OPRF protocol.
//
// Parameters:
//   - r: the blinding scalar used in Blind (32 bytes)
//   - beta: the evaluated element from the server (32 bytes)
//
// Returns:
//   - n: the unblinded element (32 bytes)
//   - error: any error that occurred
//
// The unblind operation computes:
//  1. r_inv = 1/r (scalar inversion)
//  2. n = beta^r_inv (scalar multiplication)
//
// This operation uses constant-time scalar inversion for security.
func Unblind(r []byte, beta []byte) (n []byte, err error) {
	// Validate r length
	if len(r) != ScalarBytes {
		return nil, fmt.Errorf("blind scalar must be %d bytes, got %d", ScalarBytes, len(r))
	}

	// Validate beta length
	if len(beta) != ElementBytes {
		return nil, fmt.Errorf("beta must be %d bytes, got %d", ElementBytes, len(beta))
	}

	// Decode r as scalar
	rScalar := ristretto255.NewScalar()
	if err := rScalar.Decode(r); err != nil {
		return nil, fmt.Errorf("invalid blind scalar: %w", err)
	}

	// Decode beta as element (this validates it's a valid curve point)
	betaElement := ristretto255.NewElement()
	if err := betaElement.Decode(beta); err != nil {
		return nil, fmt.Errorf("invalid beta element: %w", err)
	}

	// Compute r_inv = 1/r (constant-time scalar inversion)
	rInv := ristretto255.NewScalar()
	rInv.Invert(rScalar)

	// Compute n = beta^r_inv (scalar multiplication)
	nElement := ristretto255.NewElement()
	nElement.ScalarMult(rInv, betaElement)

	// Encode n as bytes
	n = nElement.Encode(nil)

	return n, nil
}

// Finalize computes the final OPRF output.
//
// Parameters:
//   - input: the original input to the OPRF (same as used in Blind)
//   - n: the unblinded element from Unblind (32 bytes)
//
// Returns:
//   - output: the final OPRF output (64 bytes)
//   - error: any error that occurred
//
// The finalize operation computes:
//
//	hash(len(input) || input || len(n) || n || "Finalize")
//
// where lengths are encoded as 2-byte big-endian integers (network byte order).
//
// This uses SHA-512 and outputs 64 bytes.
func Finalize(input []byte, n []byte) (output []byte, err error) {
	// Validate n length
	if len(n) != ElementBytes {
		return nil, fmt.Errorf("n must be %d bytes, got %d", ElementBytes, len(n))
	}

	// Build the hash input according to the OPRF specification
	// Format: len(input) || input || len(n) || n || "Finalize"
	// where lengths are 2-byte big-endian (network byte order)
	h := sha512.New()

	// Write len(input) as 2-byte big-endian
	inputLen := make([]byte, 2)
	binary.BigEndian.PutUint16(inputLen, uint16(len(input)))
	h.Write(inputLen)

	// Write input
	h.Write(input)

	// Write len(n) as 2-byte big-endian
	nLen := make([]byte, 2)
	binary.BigEndian.PutUint16(nLen, uint16(len(n)))
	h.Write(nLen)

	// Write n
	h.Write(n)

	// Write "Finalize" domain separation tag
	h.Write([]byte(FinalizeDST))

	// Compute final hash
	output = h.Sum(nil)

	return output, nil
}

// KeyGen generates a random OPRF private key.
//
// Returns:
//   - key: a random private key (32 bytes)
//   - error: any error that occurred
//
// The key is a random scalar in the ristretto255 scalar field.
// This uses cryptographically secure randomness.
func KeyGen() ([]byte, error) {
	// Generate 64 random bytes for FromUniformBytes
	randomBytes := make([]byte, 64)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create scalar from uniform bytes
	scalar := ristretto255.NewScalar()
	scalar.FromUniformBytes(randomBytes)

	// Encode as 32-byte key
	key := scalar.Encode(nil)

	return key, nil
}
