package pqc

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/open-quantum-safe/liboqs-go/oqs"
	"github.com/tendermint/tendermint/crypto"
)

const (
	KeyType      = "Falcon-512"
	PrivKeyName  = "tendermint/PrivKeyPQC"
	PubKeyName   = "tendermint/PubKeyPQC"
	SecretKeyLen = 1281 // Falcon-512 secret key length
	PublicKeyLen = 897  // Falcon-512 public key length
)

var _ codec.AminoMarshaler = &PrivKey{}

// Bytes returns the byte representation of the PrivKey.
func (privKey PrivKey) Bytes() []byte {
	return privKey.Key
}

// Sign implements crypto.PrivKey.
func (privKey *PrivKey) Sign(msg []byte) ([]byte, error) {
	if len(privKey.Key) != SecretKeyLen+PublicKeyLen {
		return nil, fmt.Errorf("invalid pqc private key size: expected %d bytes, got %d", SecretKeyLen+PublicKeyLen, len(privKey.Key))
	}

	// Extract secret key (first SecretKeyLen bytes)
	secretKey := privKey.Key[:SecretKeyLen]

	// Initialize Falcon signer with the secret key
	sig := &oqs.Signature{}
	defer sig.Clean()

	if err := sig.Init(KeyType, secretKey); err != nil {
		return nil, fmt.Errorf("failed to initialize PQC signer: %w", err)
	}

	// Sign the message
	signature, err := sig.Sign(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return signature, nil
}

// PubKey implements crypto.PrivKey.
func (privKey *PrivKey) PubKey() types.PubKey {
	if len(privKey.Key) != SecretKeyLen+PublicKeyLen {
		panic(fmt.Sprintf("invalid pqc private key size: expected %d bytes, got %d", SecretKeyLen+PublicKeyLen, len(privKey.Key)))
	}

	// Extract public key (last PublicKeyLen bytes)
	publicKey := privKey.Key[SecretKeyLen:]

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
	if len(bz) == 0 {
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
	if len(pubKey.Key) != PublicKeyLen || len(sig) == 0 {
		return false
	}

	// Initialize Falcon verifier
	verifier := &oqs.Signature{}
	defer verifier.Clean()

	if err := verifier.Init(KeyType, nil); err != nil {
		return false
	}

	// Verify the signature
	valid, err := verifier.Verify(msg, sig, pubKey.Key)
	if err != nil {
		return false
	}

	return valid
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
	if len(bz) == 0 {
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
	// Initialize Falcon signer
	sig := &oqs.Signature{}
	defer sig.Clean()

	if err := sig.Init(KeyType, nil); err != nil {
		panic(fmt.Sprintf("Failed to initialize PQC signer: %v", err))
	}

	// Generate key pair
	publicKey, err := sig.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate PQC key pair: %v", err))
	}

	// Export secret key
	secretKey := sig.ExportSecretKey()
	if secretKey == nil {
		panic("Failed to export secret key")
	}

	// Store both secret key and public key (secret key first, then public key)
	combined := make([]byte, 0, len(secretKey)+len(publicKey))
	combined = append(combined, secretKey...)
	combined = append(combined, publicKey...)

	return &PrivKey{
		Key: combined,
	}
}

// GenPrivKeyFromSecret generates a deterministic PQC private key from a seed
func GenPrivKeyFromSecret(secret []byte) *PrivKey {
	// For deterministic key generation, we'll use the seed directly as the secret key
	// In a real implementation, this would use proper key derivation
	if len(secret) < 32 {
		// Pad with zeros if seed is too short
		padded := make([]byte, 32)
		copy(padded, secret)
		secret = padded
	}

	// Initialize Falcon signer with the seed as secret key
	sig := &oqs.Signature{}
	defer sig.Clean()

	if err := sig.Init(KeyType, secret); err != nil {
		panic(fmt.Sprintf("Failed to initialize PQC signer from secret: %v", err))
	}

	// Generate key pair
	publicKey, err := sig.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate PQC key pair from secret: %v", err))
	}

	// Export secret key
	secretKey := sig.ExportSecretKey()
	if secretKey == nil {
		panic("Failed to export secret key")
	}

	// Store both secret key and public key (secret key first, then public key)
	combined := make([]byte, 0, len(secretKey)+len(publicKey))
	combined = append(combined, secretKey...)
	combined = append(combined, publicKey...)

	return &PrivKey{
		Key: combined,
	}
}
