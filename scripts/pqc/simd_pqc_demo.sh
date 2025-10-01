#!/usr/bin/env bash
set -euo pipefail

# Non-interactive simd demo using keyring-backend=test
# - Generates a PQC key (falcon-512)
# - Constructs a bank send tx (generate-only)
# - Signs offline
# - Validates signatures

ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
cd "$ROOT_DIR"

BIN=build/simd
if [ ! -x "$BIN" ]; then
  echo "Building simd ..."
  go build -o build/simd ./simapp/simd
fi

HOME_DIR="$ROOT_DIR/.simd_demo"
rm -rf "$HOME_DIR"
mkdir -p "$HOME_DIR"

KR="--keyring-backend=test"
H="--home=$HOME_DIR"
CHAIN_ID=demo-1
DENOM=uplume

SENDER_NAME="pqc1"
RCPT_NAME="rcpt1"

echo "== Create keys (sender: falcon-512 PQC, recipient: secp256k1) =="
$BIN keys add "$SENDER_NAME" $KR $H --algo falcon-512 --output json >/dev/null
$BIN keys add "$RCPT_NAME"   $KR $H --algo secp256k1  --output json >/dev/null

SENDER_ADDR=$($BIN keys show "$SENDER_NAME" $KR $H -a)
RCPT_ADDR=$($BIN keys show "$RCPT_NAME"   $KR $H -a)
echo "Sender:   $SENDER_NAME -> $SENDER_ADDR"
echo "Receiver: $RCPT_NAME   -> $RCPT_ADDR"

echo "== Construct unsigned tx (generate-only) =="
UNSIGNED="$HOME_DIR/unsigned.json"
$BIN tx bank send "$SENDER_ADDR" "$RCPT_ADDR" 1$DENOM \
  --fees 1$DENOM --chain-id="$CHAIN_ID" $KR $H \
  --generate-only > "$UNSIGNED"
echo "Unsigned saved: $UNSIGNED"

echo "== Offline sign with sender (PQC) =="
SIGNED="$HOME_DIR/signed.json"
$BIN tx sign "$UNSIGNED" --from "$SENDER_NAME" --chain-id="$CHAIN_ID" $KR $H \
  --offline > "$SIGNED"
echo "Signed saved:   $SIGNED"

echo "== Validate signatures =="
$BIN tx validate-signatures "$SIGNED" $KR $H | sed -n '1,200p'

echo "== Done =="

