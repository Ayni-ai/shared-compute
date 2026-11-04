#!/usr/bin/env bash
# One-time move of the model-registry signing key into Cloud KMS (2026-11).
#
# The registry's Ed25519 public key is pinned in the Android app, the installers and the
# coordinator, so the key is IMPORTED (same key, same public key) instead of replaced.
# After this script proves that KMS signs exactly like the pinned key, the local seed file
# must be deleted by the owner (last command it prints). Nothing else needs to change.
#
#   CLOUDSDK_CORE_ACCOUNT=mc@zalesgen.com infra/gcp/security/registry-key-to-kms.sh
set -euo pipefail
cd "$(dirname "$0")/../../.."

P="${GCP_PROJECT:-ayni1-507216}"; L="${GCP_REGION:-us-east1}"; R=ayni; K=registry-signer; J=registry-import-1
SEED="${SEED:-secrets/ayni-registry-signing.key}"
PUB="$(tr -d '[:space:]' < model-registry/PUBKEY)"
MODELCTL="model-registry/target/debug/sc-modelctl"
[ -x "$MODELCTL" ] || (cd model-registry && cargo build -q)
[ -f "$SEED" ] || { echo "no seed at $SEED"; exit 1; }
[ "$(wc -c < "$SEED" | tr -d ' ')" = 32 ] || { echo "$SEED is not a 32-byte Ed25519 seed"; exit 1; }

TMP="$(mktemp -d)"; chmod 700 "$TMP"
cleanup() { rm -P "$TMP"/k.der 2>/dev/null || rm -f "$TMP"/k.der; rm -rf "$TMP"; }
trap cleanup EXIT

# PKCS#8 DER for an Ed25519 private key: a fixed 16-byte header followed by the 32-byte seed.
printf '\x30\x2e\x02\x01\x00\x30\x05\x06\x03\x2b\x65\x70\x04\x22\x04\x20' > "$TMP/k.der"
cat "$SEED" >> "$TMP/k.der"

if ! gcloud kms keys describe "$K" --keyring "$R" --location "$L" --project "$P" >/dev/null 2>&1; then
  echo "==> creating import-only key $K"
  gcloud kms keys create "$K" --keyring "$R" --location "$L" --project "$P" \
    --purpose asymmetric-signing --default-algorithm ec-sign-ed25519 \
    --protection-level software --import-only --skip-initial-version-creation \
    --labels purpose=model-registry
fi
if ! gcloud kms import-jobs describe "$J" --keyring "$R" --location "$L" --project "$P" >/dev/null 2>&1; then
  echo "==> creating import job $J"
  gcloud kms import-jobs create "$J" --keyring "$R" --location "$L" --project "$P" \
    --import-method rsa-oaep-3072-sha256-aes-256 --protection-level software
fi
until [ "$(gcloud kms import-jobs describe "$J" --keyring "$R" --location "$L" --project "$P" --format='value(state)')" = ACTIVE ]; do sleep 3; done

if [ -z "$(gcloud kms keys versions list --key "$K" --keyring "$R" --location "$L" --project "$P" --filter='state=ENABLED' --format='value(name)')" ]; then
  echo "==> importing the seed (wrapped locally by gcloud; plaintext never leaves this machine)"
  CLOUDSDK_PYTHON_SITEPACKAGES=1 gcloud kms keys versions import --import-job "$J" \
    --key "$K" --keyring "$R" --location "$L" --project "$P" \
    --algorithm ec-sign-ed25519 --target-key-file "$TMP/k.der"
fi
until [ -n "$(gcloud kms keys versions list --key "$K" --keyring "$R" --location "$L" --project "$P" --filter='state=ENABLED' --format='value(name)')" ]; do sleep 3; done
V="$(gcloud kms keys versions list --key "$K" --keyring "$R" --location "$L" --project "$P" --filter='state=ENABLED' --format='value(name)' | head -1)"

echo "==> proving KMS signs like the pinned key ($PUB)"
curl -fsS https://models.ayni-ai.com/manifest.json -o "$TMP/manifest.json"
"$MODELCTL" canonical --manifest "$TMP/manifest.json" --out "$TMP/canon"
gcloud kms asymmetric-sign --version "$V" --input-file "$TMP/canon" --signature-file "$TMP/raw"
"$MODELCTL" attach-sig --manifest "$TMP/manifest.json" --raw-sig "$TMP/raw" >/dev/null
"$MODELCTL" verify --manifest "$TMP/manifest.json" --pub "$PUB"

KMS_PUB="$(gcloud kms keys versions get-public-key "$V" --output-file - 2>/dev/null | openssl pkey -pubin -outform DER 2>/dev/null | tail -c 32 | base64)"
[ "$KMS_PUB" = "$PUB" ] && echo "==> KMS public key matches model-registry/PUBKEY"

echo
echo "Done. KMS key version: $V"
echo "Grant signing only to the publisher identity, then delete the local seed:"
echo "  rm -P $SEED    # or: shred -u $SEED"
