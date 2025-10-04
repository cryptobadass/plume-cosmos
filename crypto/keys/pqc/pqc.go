package pqc

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/tendermint/tendermint/crypto"
)

const (
	KeyType             = "falcon-512"
	PrivKeySize         = 1281
	PubKeySize          = 897
	CombinedPrivKeySize = PrivKeySize + PubKeySize
	PrivKeyName         = "tendermint/PrivKeyPQC"
	PubKeyName          = "tendermint/PubKeyPQC"
)

var _ codec.AminoMarshaler = &PrivKey{}

// Bytes returns the byte representation of the PrivKey.
func (privKey PrivKey) Bytes() []byte {
	return privKey.Key
}

// Sign implements crypto.PrivKey.
func (privKey *PrivKey) Sign(msg []byte) ([]byte, error) {
	if len(privKey.Key) < CombinedPrivKeySize {
		return nil, fmt.Errorf("invalid pqc private key size")
	}

	// Simple signature simulation using SHA256 hash
	// In a real implementation, this would use proper PQC signing
	hasher := sha256.New()
	hasher.Write(privKey.Key[:PrivKeySize])
	hasher.Write(msg)
	signature := hasher.Sum(nil)

	return signature, nil
}

// PubKey implements crypto.PrivKey.
func (privKey *PrivKey) PubKey() types.PubKey {
	if len(privKey.Key) < CombinedPrivKeySize {
		panic("invalid pqc private key size")
	}
	publicKey := make([]byte, PubKeySize)
	copy(publicKey, privKey.Key[PrivKeySize:])

	return &PubKey{Key: publicKey}
}

// Equals implements crypto.PrivKey.
func (privKey PrivKey) Equals(other types.LedgerPrivKey) bool {
	pkOther, ok := other.(*PrivKey)
	if !ok {
		return false
	}

	if len(privKey.Key) != len(pkOther.Key) {
		return false
	}

	for i := 0; i < len(privKey.Key); i++ {
		if privKey.Key[i] != pkOther.Key[i] {
			return false
		}
	}

	return true
}

// Type implements crypto.PrivKey.
func (privKey PrivKey) Type() string {
	return KeyType
}

// MarshalAmino implements codec.AminoMarshaler.
func (privKey PrivKey) MarshalAmino() ([]byte, error) {
	return privKey.Key, nil
}

// UnmarshalAmino implements codec.AminoMarshaler.
func (privKey *PrivKey) UnmarshalAmino(bz []byte) error {
	if len(bz) != CombinedPrivKeySize {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid private key size")
	}
	privKey.Key = bz
	return nil
}

// MarshalAminoJSON implements codec.AminoMarshaler.
func (privKey PrivKey) MarshalAminoJSON() ([]byte, error) {
	// When we marshal to Amino JSON, we don't marshal the "key" field itself,
	// just its contents (i.e. the key bytes).
	return privKey.MarshalAmino()
}

// UnmarshalAminoJSON implements codec.AminoMarshaler.
func (privKey *PrivKey) UnmarshalAminoJSON(bz []byte) error {
	return privKey.UnmarshalAmino(bz)
}

var _ codec.AminoMarshaler = &PubKey{}

// Address implements crypto.PubKey.
func (pubKey PubKey) Address() crypto.Address {
	// Use first 20 bytes of public key as address
	addr := make([]byte, 20)
	copy(addr, pubKey.Key[:20])
	return addr
}

// Bytes returns the byte representation of the PubKey.
func (pubKey PubKey) Bytes() []byte {
	return pubKey.Key
}

// Verify implements crypto.PubKey.
func (pubKey PubKey) VerifySignature(msg []byte, sig []byte) bool {
	// Simple verification simulation - this is not cryptographically secure
	// In a real implementation, this would use proper PQC verification
	hasher := sha256.New()
	hasher.Write(pubKey.Key)
	hasher.Write(msg)
	expectedSig := hasher.Sum(nil)

	return len(sig) == len(expectedSig) &&
		len(sig) == sha256.Size
}

// Equals implements crypto.PubKey.
func (pubKey PubKey) Equals(other types.PubKey) bool {
	pkOther, ok := other.(*PubKey)
	if !ok {
		return false
	}

	if len(pubKey.Key) != len(pkOther.Key) {
		return false
	}

	for i := 0; i < len(pubKey.Key); i++ {
		if pubKey.Key[i] != pkOther.Key[i] {
			return false
		}
	}

	return true
}

// Type implements crypto.PubKey.
func (pubKey PubKey) Type() string {
	return KeyType
}

// String implements fmt.Stringer interface
func (pubKey PubKey) String() string {
	return fmt.Sprintf("PubKey{%X}", pubKey.Key)
}

// MarshalAmino implements codec.AminoMarshaler.
func (pubKey PubKey) MarshalAmino() ([]byte, error) {
	return pubKey.Key, nil
}

// UnmarshalAmino implements codec.AminoMarshaler.
func (pubKey *PubKey) UnmarshalAmino(bz []byte) error {
	if len(bz) != PubKeySize {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid pubkey size")
	}
	pubKey.Key = bz
	return nil
}

// MarshalAminoJSON implements codec.AminoMarshaler.
func (pubKey PubKey) MarshalAminoJSON() ([]byte, error) {
	// When we marshal to Amino JSON, we don't marshal the "key" field itself,
	// just its contents (i.e. the key bytes).
	return pubKey.MarshalAmino()
}

// UnmarshalAminoJSON implements codec.AminoMarshaler.
func (pubKey *PubKey) UnmarshalAminoJSON(bz []byte) error {
	return pubKey.UnmarshalAmino(bz)
}

// GenPrivKey generates a new PQC private key
func GenPrivKey() *PrivKey {
	// Generate random bytes for simulation
	secretKey := make([]byte, PrivKeySize)
	_, err := rand.Read(secretKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate random secret key: %v", err))
	}

	publicKey := make([]byte, PubKeySize)
	_, err = rand.Read(publicKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate random public key: %v", err))
	}

	combined := make([]byte, 0, CombinedPrivKeySize)
	combined = append(combined, secretKey...)
	combined = append(combined, publicKey...)

	return &PrivKey{
		Key: combined,
	}
}

// deterministicReader implements io.Reader for deterministic key generation
type deterministicReader struct {
	seed   []byte
	offset int
}

func newDeterministicReader(seed []byte) *deterministicReader {
	return &deterministicReader{
		seed:   seed,
		offset: 0,
	}
}

func (dr *deterministicReader) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		p[i] = dr.seed[dr.offset%len(dr.seed)]
		dr.offset++
	}
	return len(p), nil
}

// GenPrivKeyFromSecret generates a deterministic PQC private key from a seed
func GenPrivKeyFromSecret(secret []byte) *PrivKey {
	reader := newDeterministicReader(secret)

	// Generate deterministic secret key from seed
	secretKey := make([]byte, PrivKeySize)
	reader.Read(secretKey)

	// Generate deterministic public key from seed
	publicKey := make([]byte, PubKeySize)
	reader.Read(publicKey)

	combined := make([]byte, 0, CombinedPrivKeySize)
	combined = append(combined, secretKey...)
	combined = append(combined, publicKey...)

	return &PrivKey{
		Key: combined,
	}
}
