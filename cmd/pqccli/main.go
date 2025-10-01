package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	pqc "github.com/cosmos/cosmos-sdk/crypto/keys/pqc"
)

type genResult struct {
	PrivKeyHex   string `json:"priv_key_hex"`
	PubKeyHex    string `json:"pub_key_hex"`
	AddressHex   string `json:"address_hex"`
	PQCAddress   string `json:"pqc_address"`
	KeyType      string `json:"key_type"`
	SignatureAlg string `json:"signature_alg"`
}

func usage() {
	fmt.Fprintf(os.Stderr, "pqccli - PQC key utilities\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  pqccli gen\n")
	fmt.Fprintf(os.Stderr, "  pqccli sign <priv_hex> <msg_hex>\n")
	fmt.Fprintf(os.Stderr, "  pqccli verify <pub_hex> <msg_hex> <sig_hex>\n")
}

func mustHexDecode(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if s == "" {
		return nil, errors.New("empty hex string")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	return b, nil
}

func toHex(b []byte) string { return hex.EncodeToString(b) }

func pqcAddressFromPub(pub []byte) string {
	h := sha256.Sum256(pub)
	return pqc.PQCAddressPrefix + hex.EncodeToString(h[:20])
}

func cmdGen() error {
	priv := pqc.GenPrivKey()
	pub := priv.PubKey()
	res := genResult{
		PrivKeyHex:   toHex(priv.Bytes()),
		PubKeyHex:    toHex(pub.Bytes()),
		AddressHex:   toHex(pub.Address()),
		PQCAddress:   pqcAddressFromPub(pub.Bytes()),
		KeyType:      priv.Type(),
		SignatureAlg: pqc.KeyType,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

func cmdSign(args []string) error {
	if len(args) < 2 {
		return errors.New("sign requires <priv_hex> <msg_hex>")
	}
	privHex, msgHex := args[0], args[1]
	privBytes, err := mustHexDecode(privHex)
	if err != nil {
		return fmt.Errorf("priv_hex: %w", err)
	}
	msg, err := mustHexDecode(msgHex)
	if err != nil {
		return fmt.Errorf("msg_hex: %w", err)
	}
	priv := &pqc.PrivKey{Key: privBytes, KeyType: pqc.KeyType, Name: "CLI"}
	sig, err := priv.Sign(msg)
	if err != nil {
		return err
	}
	fmt.Println(toHex(sig))
	return nil
}

func cmdVerify(args []string) error {
	if len(args) < 3 {
		return errors.New("verify requires <pub_hex> <msg_hex> <sig_hex>")
	}
	pubHex, msgHex, sigHex := args[0], args[1], args[2]
	pubBytes, err := mustHexDecode(pubHex)
	if err != nil {
		return fmt.Errorf("pub_hex: %w", err)
	}
	msg, err := mustHexDecode(msgHex)
	if err != nil {
		return fmt.Errorf("msg_hex: %w", err)
	}
	sig, err := mustHexDecode(sigHex)
	if err != nil {
		return fmt.Errorf("sig_hex: %w", err)
	}
	pub := &pqc.PubKey{Key: pubBytes, KeyType: pqc.KeyType}
	ok := pub.VerifySignature(msg, sig)
	if !ok {
		fmt.Println("false")
		os.Exit(1)
	}
	fmt.Println("true")
	return nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	var err error
	switch cmd {
	case "gen":
		err = cmdGen()
	case "sign":
		err = cmdSign(os.Args[2:])
	case "verify":
		err = cmdVerify(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
