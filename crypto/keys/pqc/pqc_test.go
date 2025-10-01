package pqc_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/pqc"
	"github.com/stretchr/testify/require"
)

func TestSignAndValidate(t *testing.T) {
	privKey := pqc.GenPrivKey()
	pubKey := privKey.PubKey()

	msg := []byte("test message")
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)

	// Test the signature
	require.True(t, pubKey.VerifySignature(msg, sig))

	// Test address generation
	addr := pubKey.Address()
	require.Len(t, addr, 20) // Address should be 20 bytes
}

func TestPubKeyEquals(t *testing.T) {
	pqcPubKey := pqc.GenPrivKey().PubKey().(*pqc.PubKey)
	otherPubKey := pqc.GenPrivKey().PubKey().(*pqc.PubKey)

	// Test different keys
	require.False(t, pqcPubKey.Equals(otherPubKey))

	// Test same key
	require.True(t, pqcPubKey.Equals(pqcPubKey))

	// Test manually constructed key with same data
	samePubKey := &pqc.PubKey{
		Key:        pqcPubKey.Key,
		KeyType:    pqcPubKey.KeyType,
		PQCAddress: pqcPubKey.PQCAddress,
	}
	require.True(t, pqcPubKey.Equals(samePubKey))
}

func TestPrivKeyEquals(t *testing.T) {
	pqcPrivKey := pqc.GenPrivKey()
	otherPrivKey := pqc.GenPrivKey()

	// Test different keys
	require.False(t, pqcPrivKey.Equals(otherPrivKey))

	// Test same key
	require.True(t, pqcPrivKey.Equals(pqcPrivKey))

	// Test manually constructed key with same data
	samePrivKey := &pqc.PrivKey{
		Key:     pqcPrivKey.Key,
		KeyType: pqcPrivKey.KeyType,
		Name:    pqcPrivKey.Name,
	}
	require.True(t, pqcPrivKey.Equals(samePrivKey))
}

func TestGenPrivKeyFromSecret(t *testing.T) {
	secret := []byte("test secret")
	privKey := pqc.GenPrivKeyFromSecret(secret)

	require.NotNil(t, privKey)
	require.Equal(t, "falcon-512", privKey.Type())
	require.Len(t, privKey.Bytes(), pqc.PrivKeySize)

	// Test that it can sign and verify
	pubKey := privKey.PubKey()
	msg := []byte("test message")
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)
	require.True(t, pubKey.VerifySignature(msg, sig))
}

func TestPQCAddressValidation(t *testing.T) {
	privKey := pqc.GenPrivKey()
	pubKey := privKey.PubKey().(*pqc.PubKey)

	// Test valid PQC address
	require.True(t, pqc.ValidatePQCAddress(pubKey.PQCAddress))

	// Test invalid addresses
	require.False(t, pqc.ValidatePQCAddress("invalid"))
	require.False(t, pqc.ValidatePQCAddress("plume1invalid"))
	require.False(t, pqc.ValidatePQCAddress("plume2"))
	require.False(t, pqc.ValidatePQCAddress("plume2short"))
}

func TestKeyType(t *testing.T) {
	privKey := pqc.GenPrivKey()
	pubKey := privKey.PubKey()

	require.Equal(t, "falcon-512", privKey.Type())
	require.Equal(t, "falcon-512", pubKey.Type())
}

func TestBytes(t *testing.T) {
	privKey := pqc.GenPrivKey()
	pubKey := privKey.PubKey()

	require.Len(t, privKey.Bytes(), pqc.PrivKeySize)
	require.Len(t, pubKey.Bytes(), pqc.PubKeySize)
}
