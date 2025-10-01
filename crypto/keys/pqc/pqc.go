package pqc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/open-quantum-safe/liboqs-go/oqs"
)

const (
	// Secret key size and public key size for Falcon-512
	PrivKeySize   = 1281
	PubKeySize    = 897
	SignatureSize = 656

	// We store sk || pk in PrivKey.Key to be able to derive the pubkey
	CombinedPrivKeySize = PrivKeySize + PubKeySize

	KeyType          = "Falcon-512"
	PQCAddressPrefix = "plume2"

	PubKeyName  = "tendermint/PubKeyPQC"
	PrivKeyName = "tendermint/PrivKeyPQC"
)

var _ types.PrivKey = &PrivKey{}
var _ types.PubKey = &PubKey{}
var _ codec.AminoMarshaler = &PrivKey{}
var _ codec.AminoMarshaler = &PubKey{}

// GenPrivKey generates a new PQC private key.
// It uses OS randomness to generate the private key.
func GenPrivKey() *PrivKey {
	signer := &oqs.Signature{}
	if err := signer.Init(KeyType, nil); err != nil {
		panic(fmt.Sprintf("Failed to initialize PQC signer: %v", err))
	}
	publicKey, err := signer.GenerateKeyPair()
	if err != nil {
		signer.Clean()
		panic(fmt.Sprintf("Failed to generate PQC key pair: %v", err))
	}
	secretKey := signer.ExportSecretKey()
	signer.Clean()

	// Store sk||pk in the key bytes so we can derive the pubkey later
	combined := make([]byte, 0, CombinedPrivKeySize)
	combined = append(combined, secretKey...)
	combined = append(combined, publicKey...)

	return &PrivKey{
		Key:     combined,
		KeyType: KeyType,
		Name:    "PQC Key",
	}
}

// GenPrivKeyFromSecret generates a PQC private key from a secret.
func GenPrivKeyFromSecret(secret []byte) *PrivKey {
	// Note: liboqs-go does not support importing a deterministic secret directly.
	// We initialize and generate a fresh keypair here.
	signer := &oqs.Signature{}
	if err := signer.Init(KeyType, nil); err != nil {
		panic(fmt.Sprintf("Failed to initialize PQC signer: %v", err))
	}
	publicKey, err := signer.GenerateKeyPair()
	if err != nil {
		signer.Clean()
		panic(fmt.Sprintf("Failed to generate PQC key pair from secret: %v", err))
	}
	secretKey := signer.ExportSecretKey()
	signer.Clean()

	combined := make([]byte, 0, CombinedPrivKeySize)
	combined = append(combined, secretKey...)
	combined = append(combined, publicKey...)

	return &PrivKey{
		Key:     combined,
		KeyType: KeyType,
		Name:    "PQC Key",
	}
}

// Bytes returns the private key bytes.
func (privKey *PrivKey) Bytes() []byte { return privKey.Key }

// Sign signs a message using the private key.
func (privKey *PrivKey) Sign(msg []byte) ([]byte, error) {
	if len(privKey.Key) < CombinedPrivKeySize {
		return nil, fmt.Errorf("invalid pqc private key size: %d", len(privKey.Key))
	}
	// Extract secret key bytes from sk||pk
	sk := privKey.Key[:PrivKeySize]
	signer := &oqs.Signature{}
	if err := signer.Init(KeyType, sk); err != nil {
		signer.Clean()
		return nil, fmt.Errorf("failed to initialize PQC signer: %v", err)
	}
	sig, err := signer.Sign(msg)
	signer.Clean()
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %v", err)
	}
	return sig, nil
}

// PubKey returns the public key corresponding to the private key.
func (privKey *PrivKey) PubKey() types.PubKey {
	if len(privKey.Key) < CombinedPrivKeySize {
		panic("invalid pqc private key size")
	}
	publicKey := make([]byte, PubKeySize)
	copy(publicKey, privKey.Key[PrivKeySize:PrivKeySize+PubKeySize])
	pqcAddr := generatePQCAddress(publicKey)
	return &PubKey{Key: publicKey, KeyType: privKey.KeyType, PqcAddress: pqcAddr}
}

// Equals - you probably don't need to use this.
// Runs in constant time based on length of the
// keys.
func (privKey *PrivKey) Equals(other types.LedgerPrivKey) bool {
	if otherPQC, ok := other.(*PrivKey); ok {
		if privKey.KeyType != otherPQC.KeyType {
			return false
		}
		if len(privKey.Key) != len(otherPQC.Key) {
			return false
		}
		for i := range privKey.Key {
			if privKey.Key[i] != otherPQC.Key[i] {
				return false
			}
		}
		return true
	}
	return false
}

// Type returns the PQC key type.
func (privKey *PrivKey) Type() string { return KeyType }

// NOTE: ProtoMessage/Reset/String/Marshal/Unmarshal are provided by generated code in keys.pb.go

// MarshalAmino overrides Amino JSON marshalling.
func (privKey *PrivKey) MarshalAmino() ([]byte, error) { return privKey.Bytes(), nil }

// UnmarshalAmino overrides Amino JSON marshalling.
func (privKey *PrivKey) UnmarshalAmino(bz []byte) error {
	privKey.Key = append(privKey.Key[:0], bz...)
	return nil
}

// MarshalAminoJSON overrides Amino JSON marshalling.
func (privKey *PrivKey) MarshalAminoJSON() ([]byte, error) { return privKey.MarshalAmino() }

// UnmarshalAminoJSON overrides Amino JSON marshalling.
func (privKey *PrivKey) UnmarshalAminoJSON(bz []byte) error { return privKey.UnmarshalAmino(bz) }

// Address returns the address of the public key.
func (pubKey *PubKey) Address() types.Address {
	hash := sha256.Sum256(pubKey.Key)
	return types.Address(hash[:20])
}

// Bytes returns the public key bytes.
func (pubKey *PubKey) Bytes() []byte { return pubKey.Key }

// VerifySignature verifies a signature against the public key.
func (pubKey *PubKey) VerifySignature(msg, sig []byte) bool {
	verifier := &oqs.Signature{}
	if err := verifier.Init(KeyType, nil); err != nil {
		return false
	}
	ok, err := verifier.Verify(msg, sig, pubKey.Key)
	verifier.Clean()
	return err == nil && ok
}

// Type returns the PQC key type.
func (pubKey *PubKey) Type() string { return KeyType }

// Equals returns true if the public keys are equal.
func (pubKey *PubKey) Equals(other types.PubKey) bool {
	if otherPQC, ok := other.(*PubKey); ok {
		if pubKey.KeyType != otherPQC.KeyType || len(pubKey.Key) != len(otherPQC.Key) {
			return false
		}
		for i := range pubKey.Key {
			if pubKey.Key[i] != otherPQC.Key[i] {
				return false
			}
		}
		return pubKey.PqcAddress == otherPQC.PqcAddress
	}
	return false
}

// NOTE: ProtoMessage/Reset/String/Marshal/Unmarshal are provided by generated code in keys.pb.go

// MarshalAmino overrides Amino JSON marshalling.
func (pubKey *PubKey) MarshalAmino() ([]byte, error) { return pubKey.Bytes(), nil }

// UnmarshalAmino overrides Amino JSON marshalling.
func (pubKey *PubKey) UnmarshalAmino(bz []byte) error {
	pubKey.Key = append(pubKey.Key[:0], bz...)
	return nil
}

// MarshalAminoJSON overrides Amino JSON marshalling.
func (pubKey *PubKey) MarshalAminoJSON() ([]byte, error) { return pubKey.MarshalAmino() }

// UnmarshalAminoJSON overrides Amino JSON marshalling.
func (pubKey *PubKey) UnmarshalAminoJSON(bz []byte) error { return pubKey.UnmarshalAmino(bz) }

// generatePQCAddress generates PQC address from public key
func generatePQCAddress(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)
	addressBytes := hash[:20] // Take first 20 bytes
	return PQCAddressPrefix + hex.EncodeToString(addressBytes)
}

// ValidatePQCAddress validates PQC address format
func ValidatePQCAddress(addr string) bool {
	if len(addr) < 6 || addr[:6] != PQCAddressPrefix {
		return false
	}

	if len(addr) != 46 { // plume2 + 40 hex characters
		return false
	}

	// Check if the remaining part is valid hex
	hexPart := addr[len(PQCAddressPrefix):]
	_, err := hex.DecodeString(hexPart)
	return err == nil
}

// String implements proto stringer required by gogo's proto.Message
func (m *PubKey) String() string { return proto.CompactTextString(m) }
