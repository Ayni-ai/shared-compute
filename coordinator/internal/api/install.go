package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// defaultRegistryPubKey is the model registry's Ed25519 verifying key (model-registry/PUBKEY).
const defaultRegistryPubKey = "p7UUs6aCFebGUfSvFV5Wczh7kYBCEW2tDRO+dV0IpDM="

//go:embed provider-sha256.txt
var providerSHA256 string

var shaLineRE = regexp.MustCompile(`^([0-9a-f]{64})\s+(provider-daemon-[a-z0-9.-]+)$`)

// pinnedSHACases renders provider-sha256.txt as shell case arms for expected_sha().
func pinnedSHACases() string {
	var b strings.Builder
	for _, line := range strings.Split(providerSHA256, "\n") {
		m := shaLineRE.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		fmt.Fprintf(&b, "    %s) echo %s ;;\n", m[2], m[1])
	}
	return b.String()
}

// GET /install/provider.sh — a self-contained installer the "Share your computer"
// page hands out. It downloads a prebuilt provider-daemon for the host (falling
// back to a source build only on request), verifies it against the pinned SHA-256,
// lets the daemon fetch and verify the model from the signed registry, and connects
// to this coordinator. The account's registration token is passed in the environment,
// never baked into the script.
func (s *Server) handleInstallScript(w http.ResponseWriter, _ *http.Request) {
	apiBase := s.cfg.AuthCallbackBase // the API origin, e.g. https://api.ayni-ai.com
	if apiBase == "" {
		apiBase = "https://api.ayni-ai.com"
	}
	ws := strings.Replace(apiBase, "http", "ws", 1) + "/ws/provider"
	model := "qwen2.5-0.5b-instruct-q4_k_m"
	modelURL := strings.TrimRight(s.cfg.ManifestURL, "/")
	if modelURL == "" {
		modelURL = "https://models.ayni-ai.com"
	}
	registryPub := s.cfg.RegistryPubKey
	if registryPub == "" {
		registryPub = defaultRegistryPubKey
	}
	repo := "https://github.com/Ayni-ai/shared-compute"
	appBase := s.cfg.AppURL
	if appBase == "" {
		appBase = "https://app.ayni-ai.com"
	}

	script := fmt.Sprintf(`#!/usr/bin/env bash
# Ayni provider installer. Runs your machine as a compute provider.
set -euo pipefail

: "${SC_REGISTRATION_TOKEN:?set SC_REGISTRATION_TOKEN (from the Share your computer page)}"
COORD_WS="${SC_COORDINATOR_URL:-%s}"
MODEL="%s"
# The provider downloads the model itself from the signed registry and checks every
# file's SHA-256 against a manifest signed with the registry key pinned here.
MANIFEST_URL="%s"
REGISTRY_PUBKEY="%s"
DIR="${AYNI_DIR:-$HOME/.ayni}"
mkdir -p "$DIR/models"
# Installs before 2026-11 kept the model at models/<id>.gguf; the verified store
# keeps it at models/<id>/<file>. Move it so the store re-checks it instead of re-downloading.
if [ -f "$DIR/models/$MODEL.gguf" ] && [ ! -d "$DIR/models/$MODEL" ]; then
  mkdir -p "$DIR/models/$MODEL" && mv "$DIR/models/$MODEL.gguf" "$DIR/models/$MODEL/$MODEL.gguf"
fi

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"; [ "$arch" = "x86_64" ] && arch="x64"; [ "$arch" = "aarch64" ] && arch="arm64"
asset="provider-daemon-${os}-${arch}"
BIN="$DIR/provider-daemon"

# SHA-256 of each published provider binary, pinned in the coordinator's source
# (coordinator/internal/api/provider-sha256.txt) and changed only by a reviewed commit.
expected_sha() {
  case "$1" in
%s    *) echo "" ;;
  esac
}
sha_of() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

WANT="$(expected_sha "$asset")"
if [ -x "$BIN" ] && [ -n "$WANT" ] && [ "$(sha_of "$BIN")" != "$WANT" ]; then
  echo "==> the installed provider-daemon does not match the pinned release; replacing it"
  rm -f "$BIN"
fi
if [ ! -x "$BIN" ]; then
  if [ -z "$WANT" ]; then
    if [ "${SC_BUILD_FROM_SOURCE:-}" != "1" ]; then
      echo "==> there is no verified prebuilt provider for $os/$arch." >&2
      echo "    Set SC_BUILD_FROM_SOURCE=1 to build it from source instead (needs Rust and CMake)." >&2
      exit 1
    fi
    echo "==> building provider-daemon from source"
    [ -d "$DIR/src" ] || git clone --depth 1 %s "$DIR/src"
    ( cd "$DIR/src/provider-core" && cargo build --release --bin provider-daemon --features llama )
    cp "$DIR/src/provider-core/target/release/provider-daemon" "$BIN"
  else
    url="%s/releases/download/provider-latest/${asset}"
    echo "==> fetching $asset"
    curl -fSL --retry 3 --retry-delay 2 "$url" -o "$BIN.part"
    GOT="$(sha_of "$BIN.part")"
    if [ "$GOT" != "$WANT" ]; then
      rm -f "$BIN.part"
      echo "==> checksum mismatch for $asset: refusing to run it." >&2
      echo "    expected $WANT" >&2
      echo "    got      $GOT" >&2
      exit 1
    fi
    mv "$BIN.part" "$BIN"
    chmod +x "$BIN"
    echo "==> verified $asset (sha256 $WANT)"
  fi
fi

# Pin the coordinator's Workload Manifest signing key so this node only runs
# jobs the coordinator actually signed (fail closed, like the phone app).
KEY_JSON="$(curl -fsSL "%s/v1/manifest-key" 2>/dev/null || true)"
SIGNER="$(printf '%%s' "$KEY_JSON" | sed -n 's/.*"signer_id":"\([^"]*\)".*/\1/p')"
PUBKEY="$(printf '%%s' "$KEY_JSON" | sed -n 's/.*"public_key":"\([^"]*\)".*/\1/p')"
VERIFY_KEY="${SC_MANIFEST_VERIFY_KEY:-}"
if [ -z "$VERIFY_KEY" ] && [ -n "$SIGNER" ] && [ -n "$PUBKEY" ]; then VERIFY_KEY="$SIGNER:$PUBKEY"; fi
[ -n "$VERIFY_KEY" ] && echo "==> manifest signing key pinned: ${VERIFY_KEY%%%%:*}"

echo "==> connecting to $COORD_WS"
echo ""
echo "    Ayni provider is starting. Leave this window open; press Ctrl-C to stop sharing."
echo "    Watch it appear at %s/share/ — the page confirms the device within seconds."
echo "    Set SC_VERBOSE=1 to see the model runtime's own log lines."
echo ""

# The model runtime prints a great deal of low-level detail (tensor loading,
# Metal/CUDA kernels). Hide it unless the operator asks, keeping the daemon's
# own lines (connecting / registered / job finished / errors).
quiet() {
  if [ -n "${SC_VERBOSE:-}" ]; then cat; else
    grep --line-buffered -vE '^(ggml_|llama_|print_info|load(_tensors)?:|create_tensor|done_getting|sched_reserve|graph_reserve|resolve_fused|init_tokenizer|set_abort|~llama|\.+$)'
  fi
}

env \
  SC_COORDINATOR_URL="$COORD_WS" \
  SC_REGISTRATION_TOKEN="$SC_REGISTRATION_TOKEN" \
  SC_MODEL="$MODEL" \
  SC_MANIFEST_URL="$MANIFEST_URL" \
  SC_REGISTRY_PUBKEY="$REGISTRY_PUBKEY" \
  SC_MODEL_DIR="$DIR/models" \
  SC_BACKEND="${SC_BACKEND:-llama}" \
  SC_IDENTITY_PATH="$DIR/identity.key" \
  SC_MANIFEST_VERIFY_KEY="$VERIFY_KEY" \
  SC_REQUIRE_MANIFEST="${SC_REQUIRE_MANIFEST:-1}" \
  "$BIN" 2>&1 | quiet
`, ws, model, modelURL, registryPub, pinnedSHACases(), repo, repo, apiBase, appBase)

	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(script))
}
