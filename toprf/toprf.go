// Package toprf implements the Threshold Oblivious Pseudorandom Function (TOPRF)
// protocol using the 3hashTDH construction.
//
// This implementation follows the TOPPSS paper (https://eprint.iacr.org/2017/363)
// and the 3HashTDH protocol from Gu et al. 2024 (https://eprint.iacr.org/2024/1455).
//
// # What is Threshold OPRF?
//
// The threshold OPRF allows splitting an OPRF key across multiple servers using
// Shamir secret sharing, such that any threshold number of servers can cooperate
// to evaluate the OPRF, but fewer servers learn nothing about the result.
//
// This provides:
//   - Resilience: System works even if some servers are offline (up to n-threshold)
//   - Security: Secret key is never fully reconstructed, remains distributed
//   - Privacy: Client's input remains oblivious to all servers
//
// # Protocol Flow
//
// The basic threshold OPRF protocol:
//
//  1. Setup: Generate n shares of a secret key using CreateShares(secret, n, threshold)
//     Distribute one share to each of n servers
//
//  2. Client: Blind input using oprf.Blind() (same as basic OPRF)
//     Send blinded element alpha to threshold servers
//
//  3. Servers: Each server evaluates using Evaluate(share, alpha, indexes)
//     Returns partial evaluation (Part) to client
//
//  4. Client: Combine partial evaluations using ThresholdCombine(parts)
//     Then unblind and finalize using oprf.Unblind() and oprf.Finalize()
//
// # Usage Example
//
//	// Setup: Create shares for 5 servers with threshold 3
//	secret, _ := oprf.KeyGen()
//	secretScalar := ristretto255.NewScalar()
//	secretScalar.Decode(secret)
//	shares, _ := toprf.CreateShares(secretScalar, 5, 3)
//	// Distribute shares[i] to server i
//
//	// Client: Blind input
//	input := []byte("secret input")
//	r, alpha, _ := oprf.Blind(input, nil)
//
//	// Servers 1, 2, 3 evaluate (any 3 of 5 servers)
//	indexes := []uint8{1, 2, 3}
//	part1, _ := toprf.Evaluate(shares[0], alpha, indexes)
//	part2, _ := toprf.Evaluate(shares[1], alpha, indexes)
//	part3, _ := toprf.Evaluate(shares[2], alpha, indexes)
//
//	// Client: Combine partial evaluations
//	beta, _ := toprf.ThresholdCombine([][]byte{part1, part2, part3})
//
//	// Client: Unblind and finalize
//	n, _ := oprf.Unblind(r, beta)
//	output, _ := oprf.Finalize(input, n)
//	// output is the final threshold OPRF result
//
// # Security Model
//
// The 3HashTDH protocol provides security even when all threshold servers are
// compromised. The protocol adds an extra layer using zero-sharing to ensure
// that learning all server keys does not compromise past OPRF outputs.
//
// Use ThreeHashTDH() instead of Evaluate() for this enhanced security model.
//
// # Compatibility
//
// This implementation is byte-for-byte compatible with liboprf's threshold
// OPRF implementation and follows standard Shamir secret sharing on ristretto255.
package toprf

import (
	"crypto/rand"
	"encoding/binary"
	"errors"

	"github.com/gtank/ristretto255"
	"golang.org/x/crypto/blake2b"
)

// Constants for threshold OPRF
const (
	// ShareBytes is the size of a serialized share (1 byte index + 32 byte value)
	ShareBytes = 33

	// PartBytes is the size of a serialized partial evaluation (32 byte element + 1 byte index)
	PartBytes = 33

	// ScalarBytes is the size of a ristretto255 scalar
	ScalarBytes = 32

	// ElementBytes is the size of a ristretto255 element
	ElementBytes = 32
)

// Share represents a Shamir secret share with an index and scalar value.
// Shares are used to distribute a secret key across multiple parties in a
// threshold scheme.
//
// Each share consists of:
//   - Index: The participant's identifier (1-based, 1..n)
//   - Value: The secret share as a ristretto255 scalar
//
// Shares are created using CreateShares() and can be marshaled for
// transmission or storage.
type Share struct {
	Index uint8
	Value *ristretto255.Scalar
}

// MarshalBinary encodes a Share into bytes for transmission or storage.
// Format: [index:1 byte][value:32 bytes] = 33 bytes total
func (s *Share) MarshalBinary() ([]byte, error) {
	if s.Value == nil {
		return nil, errors.New("toprf: share value is nil")
	}

	data := make([]byte, ShareBytes)
	data[0] = s.Index
	copy(data[1:], s.Value.Encode(nil))
	return data, nil
}

// UnmarshalBinary decodes a Share from bytes.
// Expects data to be exactly ShareBytes (33 bytes).
func (s *Share) UnmarshalBinary(data []byte) error {
	if len(data) != ShareBytes {
		return errors.New("toprf: invalid share length")
	}

	s.Index = data[0]
	s.Value = ristretto255.NewScalar()
	if err := s.Value.Decode(data[1:]); err != nil {
		return err
	}
	return nil
}

// Part represents a partial evaluation result from one server in the
// threshold OPRF protocol.
//
// Each part consists of:
//   - Index: The server's identifier (matches the share index used for evaluation)
//   - Element: The partial OPRF evaluation as a ristretto255 element
//
// Parts are returned by Evaluate() and ThreeHashTDH(), and are combined
// using ThresholdCombine() to produce the final evaluation result.
type Part struct {
	Index   uint8
	Element *ristretto255.Element
}

// MarshalBinary encodes a Part into bytes for transmission.
// Format: [index:1 byte][element:32 bytes] = 33 bytes total
func (p *Part) MarshalBinary() ([]byte, error) {
	if p.Element == nil {
		return nil, errors.New("toprf: part element is nil")
	}

	data := make([]byte, PartBytes)
	data[0] = p.Index
	copy(data[1:], p.Element.Encode(nil))
	return data, nil
}

// UnmarshalBinary decodes a Part from bytes.
// Expects data to be exactly PartBytes (33 bytes).
func (p *Part) UnmarshalBinary(data []byte) error {
	if len(data) != PartBytes {
		return errors.New("toprf: invalid part length")
	}

	p.Index = data[0]
	p.Element = ristretto255.NewElement()
	if err := p.Element.Decode(data[1:]); err != nil {
		return err
	}
	return nil
}

// scalarFromUint8 creates a ristretto255 scalar from a uint8 value
func scalarFromUint8(v uint8) *ristretto255.Scalar {
	s := ristretto255.NewScalar()
	var buf [32]byte
	buf[0] = v
	s.Decode(buf[:])
	return s
}

// lcoeff computes the Lagrange coefficient for interpolation at point x.
// It computes: ∏(x - peers[j]) / ∏(index - peers[j]) for j != index
// This is used in Lagrange polynomial interpolation.
func lcoeff(index, x uint8, peers []uint8) *ristretto255.Scalar {
	xScalar := scalarFromUint8(x)
	iScalar := scalarFromUint8(index)
	dividend := scalarFromUint8(1)
	divisor := scalarFromUint8(1)

	for _, peer := range peers {
		if peer == index {
			continue
		}

		peerScalar := scalarFromUint8(peer)

		// dividend *= (x - peer)
		tmp := ristretto255.NewScalar()
		tmp.Subtract(peerScalar, xScalar) // Note: Subtract(a,b) computes b-a
		dividend.Multiply(dividend, tmp)

		// divisor *= (index - peer)
		tmp = ristretto255.NewScalar()
		tmp.Subtract(peerScalar, iScalar) // Note: Subtract(a,b) computes b-a
		divisor.Multiply(divisor, tmp)
	}

	// result = dividend / divisor = dividend * divisor^(-1)
	divisor.Invert(divisor)
	result := ristretto255.NewScalar()
	result.Multiply(dividend, divisor)

	return result
}

// coeff computes the Lagrange coefficient for f(0), which is used when
// reconstructing the secret from shares.
func coeff(index uint8, peers []uint8) *ristretto255.Scalar {
	return lcoeff(index, 0, peers)
}

// interpolate reconstructs a polynomial value at point x using Lagrange interpolation.
// Given shares that are evaluations of a polynomial f at different points,
// this computes f(x).
// InterpolateScalar performs Lagrange interpolation at point x using the given shares.
// Returns the scalar value at x.
// This is used internally but also exported for use by the DKG package.
func InterpolateScalar(x uint8, shares []Share) (*ristretto255.Scalar, error) {
	if len(shares) == 0 {
		return nil, errors.New("toprf: no shares provided")
	}

	result := ristretto255.NewScalar()
	result.Decode([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	// Extract indexes from shares
	indexes := make([]uint8, len(shares))
	for i, share := range shares {
		indexes[i] = share.Index
	}

	// For each share, compute l_i(x) * share.value and add to result
	for _, share := range shares {
		l := lcoeff(share.Index, x, indexes)
		term := ristretto255.NewScalar()
		term.Multiply(l, share.Value)
		result.Add(result, term)
	}

	return result, nil
}

// interpolate is a legacy wrapper for backward compatibility
func interpolate(x uint8, shares []Share) *ristretto255.Scalar {
	result, _ := InterpolateScalar(x, shares)
	return result
}

// CreateShares splits a secret into n shares using Shamir's secret sharing scheme.
// Any threshold number of shares can reconstruct the secret, but fewer reveal nothing.
// The secret is the constant term (f(0)) of a random polynomial of degree threshold-1.
//
// Shares are indexed from 1 to n (not 0 to n-1).
func CreateShares(secret *ristretto255.Scalar, n, threshold uint8) ([]Share, error) {
	if threshold < 1 || n < threshold {
		return nil, errors.New("toprf: invalid threshold parameters")
	}
	if threshold > n {
		return nil, errors.New("toprf: threshold cannot exceed n")
	}

	// Generate random polynomial coefficients a[0], a[1], ..., a[threshold-2]
	// f(x) = secret + a[0]*x + a[1]*x^2 + ... + a[threshold-2]*x^(threshold-1)
	coeffs := make([]*ristretto255.Scalar, threshold-1)
	for i := range coeffs {
		coeffs[i] = ristretto255.NewScalar()
		var randBytes [64]byte
		if _, err := rand.Read(randBytes[:]); err != nil {
			return nil, err
		}
		coeffs[i].FromUniformBytes(randBytes[:])
	}

	// Create shares: f(i) for i in 1..n
	shares := make([]Share, n)
	for i := uint8(1); i <= n; i++ {
		shares[i-1].Index = i
		shares[i-1].Value = ristretto255.NewScalar()

		// Start with f(i) = secret
		shares[i-1].Value.Add(shares[i-1].Value, secret)

		// Compute x = i as scalar
		x := scalarFromUint8(i)

		// Add terms: a[j] * x^(j+1)
		for j := 0; j < int(threshold-1); j++ {
			// Compute term = a[j] * x
			term := ristretto255.NewScalar()
			term.Multiply(coeffs[j], x)

			// Multiply by x for j more times to get x^(j+1)
			for exp := 0; exp < j; exp++ {
				term.Multiply(term, x)
			}

			// Add to share value
			shares[i-1].Value.Add(shares[i-1].Value, term)
		}
	}

	return shares, nil
}

// Evaluate performs a threshold OPRF evaluation using a key share.
// It computes the Lagrange coefficient for the share based on the list of
// all participating peer indexes, then evaluates the blinded element.
//
// The result is a Part containing the partial evaluation and the share's index.
func Evaluate(share Share, blinded []byte, indexes []uint8) ([]byte, error) {
	if len(blinded) != ElementBytes {
		return nil, errors.New("toprf: invalid blinded element length")
	}

	// Compute Lagrange coefficient for this share
	c := coeff(share.Index, indexes)

	// Multiply share value by coefficient
	adjustedKey := ristretto255.NewScalar()
	adjustedKey.Multiply(share.Value, c)

	// Use basic OPRF evaluate - need to import oprf package
	// For now, implement directly here
	alpha := ristretto255.NewElement()
	if err := alpha.Decode(blinded); err != nil {
		return nil, err
	}

	// Compute beta = alpha^adjustedKey
	beta := ristretto255.NewElement()
	beta.ScalarMult(adjustedKey, alpha)

	// Create Part with index and element
	part := Part{
		Index:   share.Index,
		Element: beta,
	}

	return part.MarshalBinary()
}

// ThresholdCombine combines partial evaluations from threshold servers.
// This version assumes Lagrange coefficients were already applied during evaluation.
// It simply adds all the partial results together.
func ThresholdCombine(responses [][]byte) ([]byte, error) {
	if len(responses) == 0 {
		return nil, errors.New("toprf: no responses to combine")
	}
	if len(responses) > 255 {
		return nil, errors.New("toprf: too many responses")
	}

	// Parse all responses into Parts and sort by index
	parts := make([]Part, len(responses))
	for i, resp := range responses {
		if err := parts[i].UnmarshalBinary(resp); err != nil {
			return nil, err
		}
	}

	// Sort parts by index (simple bubble sort for small arrays)
	for i := 0; i < len(parts)-1; i++ {
		for j := i + 1; j < len(parts); j++ {
			if parts[j].Index < parts[i].Index {
				parts[i], parts[j] = parts[j], parts[i]
			}
		}
	}

	// Add all elements together (NewElement() returns identity element)
	result := ristretto255.NewElement()
	for _, part := range parts {
		result.Add(result, part.Element)
	}

	return result.Encode(nil), nil
}

// ThreeHashTDH implements the 3HashTDH protocol from Gu et al. 2024.
// This provides threshold OPRF evaluation with security against compromise of all servers.
//
// Parameters:
//   - k: share of the secret key
//   - z: random zero-sharing (share of 0 for threshold security)
//   - alpha: blinded element from client
//   - ssid: session-specific identifier (must be same for all participants)
//
// The function computes: beta = alpha^k + H(ssid||alpha)^z
func ThreeHashTDH(k, z Share, alpha, ssid []byte) ([]byte, error) {
	if len(alpha) != ElementBytes {
		return nil, errors.New("toprf: invalid alpha length")
	}

	// Evaluate alpha with key share k: beta = alpha^k
	alphaElement := ristretto255.NewElement()
	if err := alphaElement.Decode(alpha); err != nil {
		return nil, err
	}

	beta := ristretto255.NewElement()
	beta.ScalarMult(k.Value, alphaElement)

	// Hash ssid and alpha using BLAKE2b
	// Format: htons(len(ssid)) || ssid || alpha
	h, err := blake2b.New512(nil)
	if err != nil {
		return nil, err
	}

	// Write length prefix in network byte order (big-endian)
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(ssid)))
	h.Write(lenBuf)
	h.Write(ssid)
	h.Write(alpha)

	hash := h.Sum(nil) // 64 bytes

	// Hash-to-curve: convert hash to ristretto255 point
	point := ristretto255.NewElement()
	point.FromUniformBytes(hash)

	// Evaluate point with zero-share z: h2 = point^z
	h2 := ristretto255.NewElement()
	h2.ScalarMult(z.Value, point)

	// Add both evaluations: beta = beta + h2
	beta.Add(beta, h2)

	// Return as Part (index + element)
	part := Part{
		Index:   k.Index,
		Element: beta,
	}

	return part.MarshalBinary()
}
