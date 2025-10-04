package main

import (
	"fmt"
	"log"

	"github.com/cosmos/cosmos-sdk/crypto/keys/pqc"
)

func main() {
	fmt.Println("Testing PQC key generation and signing...")
	
	// Generate a PQC private key
	privKey := pqc.GenPrivKey()
	fmt.Printf("Generated private key size: %d bytes\n", len(privKey.Key))
	
	// Get public key
	pubKey := privKey.PubKey()
	fmt.Printf("Generated public key size: %d bytes\n", len(pubKey.Bytes()))
	
	// Test message
	msg := []byte("Hello, PQC World!")
	fmt.Printf("Testing message: %s\n", string(msg))
	
	// Sign the message
	signature, err := privKey.Sign(msg)
	if err != nil {
		log.Fatalf("Failed to sign message: %v", err)
	}
	fmt.Printf("Signature length: %d bytes\n", len(signature))
	
	// Verify the signature
	valid := pubKey.VerifySignature(msg, signature)
	fmt.Printf("Signature verification: %t\n", valid)
	
	if valid {
		fmt.Println("✅ PQC signing and verification successful!")
	} else {
		fmt.Println("❌ PQC signature verification failed!")
	}
}
