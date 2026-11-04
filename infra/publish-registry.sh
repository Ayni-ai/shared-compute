#!/usr/bin/env bash
# Build + sign the model manifest and upload it (plus any new model files) to the R2
# bucket behind models.ayni-ai.com.
#
# Signing happens in Cloud KMS (key registry-signer, imported 2026-11 from the original
# Ed25519 seed so the pinned public key did not change). No signing key exists on disk.
# Run it from Cloud Shell or Cloud Build (infra/gcp/registry-publish.cloudbuild.yaml),
# not from a laptop.
#
# Prereqs:
#   - roles/cloudkms.signer on REGISTRY_KMS_KEY for the caller
#   - rclone installed
#   - either an rclone remote named "r2", OR these env vars:
#       R2_ACCOUNT_ID R2_ACCESS_KEY_ID R2_SECRET_ACCESS_KEY
#
# Usage:
#   ./infra/publish-registry.sh add  <model_id> <arch> <quant> <hw_class> <ctx> <file.gguf>...
#   ./infra/publish-registry.sh push
set -euo pipefail
cd "$(dirname "$0")/.."

BUCKET="${SC_R2_BUCKET:-models}"
STAGE="${SC_REGISTRY_STAGE:-.registry}"
REGISTRY_KMS_KEY="${REGISTRY_KMS_KEY:-projects/ayni1-507216/locations/us-east1/keyRings/ayni/cryptoKeys/registry-signer/cryptoKeyVersions/1}"
# Break-glass only: sign with a local seed file. Never the default.
LOCAL_KEY="${ALLOW_LOCAL_REGISTRY_KEY:-}"
MODELCTL="${MODELCTL:-model-registry/target/release/sc-modelctl}"
[[ -x "$MODELCTL" ]] || MODELCTL="model-registry/target/debug/sc-modelctl"
[[ -x "$MODELCTL" ]] || { echo "build sc-modelctl first: (cd model-registry && cargo build)"; exit 1; }

# Configure an rclone "r2" remote via env vars unless one already exists in the config.
setup_rclone() {
  command -v rclone >/dev/null || { echo "install rclone (brew install rclone)"; exit 1; }
  if rclone listremotes 2>/dev/null | grep -qx 'r2:'; then
    return
  fi
  : "${R2_ACCOUNT_ID:?set R2_ACCOUNT_ID}" \
    "${R2_ACCESS_KEY_ID:?set R2_ACCESS_KEY_ID}" \
    "${R2_SECRET_ACCESS_KEY:?set R2_SECRET_ACCESS_KEY}"
  export RCLONE_CONFIG_R2_TYPE=s3
  export RCLONE_CONFIG_R2_PROVIDER=Cloudflare
  export RCLONE_CONFIG_R2_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
  export RCLONE_CONFIG_R2_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
  export RCLONE_CONFIG_R2_ENDPOINT="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"
  export RCLONE_CONFIG_R2_REGION=auto
  # Object-scoped tokens can't CreateBucket; the bucket already exists.
  export RCLONE_CONFIG_R2_NO_CHECK_BUCKET=true
}

case "${1:-}" in
  add)
    shift
    model_id="$1"; arch="$2"; quant="$3"; hw="$4"; ctx="$5"; shift 5
    mkdir -p "$STAGE/$model_id"
    for f in "$@"; do cp -n "$f" "$STAGE/$model_id/"; done
    "$MODELCTL" add --manifest "$STAGE/manifest.json" --model-id "$model_id" \
      --arch "$arch" --quant "$quant" --hw-class "$hw" --ctx "$ctx" "$@"
    echo "staged. run: $0 push"
    ;;
  push)
    if [[ -n "$LOCAL_KEY" ]]; then
      echo "WARNING: signing with the local key $LOCAL_KEY (break-glass)" >&2
      "$MODELCTL" sign --manifest "$STAGE/manifest.json" --key "$LOCAL_KEY"
    else
      "$MODELCTL" canonical --manifest "$STAGE/manifest.json" --out "$STAGE/.canonical"
      gcloud kms asymmetric-sign --version "$REGISTRY_KMS_KEY" \
        --input-file "$STAGE/.canonical" --signature-file "$STAGE/.raw-sig"
      "$MODELCTL" attach-sig --manifest "$STAGE/manifest.json" --raw-sig "$STAGE/.raw-sig"
      rm -f "$STAGE/.canonical" "$STAGE/.raw-sig"
    fi
    # Never publish a manifest the pinned key would reject.
    "$MODELCTL" verify --manifest "$STAGE/manifest.json" --pub model-registry/PUBKEY
    setup_rclone
    # model files first (immutable content), then manifest + sig (the switch)
    rclone copy --progress --exclude 'manifest.json*' --exclude '.*' "$STAGE" "r2:$BUCKET"
    rclone copyto "$STAGE/manifest.json"     "r2:$BUCKET/manifest.json"
    rclone copyto "$STAGE/manifest.json.sig" "r2:$BUCKET/manifest.json.sig"
    echo
    echo "published to r2:$BUCKET  ->  https://models.ayni-ai.com/manifest.json"
    ;;
  *)
    echo "usage: $0 add <model_id> <arch> <quant> <hw_class> <ctx> <file.gguf>... | push"
    exit 2
    ;;
esac
