package api

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mcastroarroyo/shared-compute/coordinator/internal/config"
)

func TestInstallScriptRenders(t *testing.T) {
	s := &Server{cfg: config.Config{
		AuthCallbackBase: "https://api.example.com",
		ManifestURL:      "https://models.example.com",
	}}
	rec := httptest.NewRecorder()
	s.handleInstallScript(rec, httptest.NewRequest("GET", "/install/provider.sh", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()

	// the Go template must not leak doubled braces
	if strings.Contains(body, "${{") || strings.Contains(body, "}}") {
		t.Fatalf("unrendered braces in script:\n%s", body)
	}
	for _, want := range []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		`COORD_WS="${SC_COORDINATOR_URL:-wss://api.example.com/ws/provider}"`,
		`MANIFEST_URL="https://models.example.com"`,
		`REGISTRY_PUBKEY="p7UUs6aCFebGUfSvFV5Wczh7kYBCEW2tDRO+dV0IpDM="`,
		`SC_MANIFEST_URL="$MANIFEST_URL"`,
		`SC_REGISTRY_PUBKEY="$REGISTRY_PUBKEY"`,
		"releases/download/provider-latest/${asset}",
		`SC_REGISTRATION_TOKEN="$SC_REGISTRATION_TOKEN"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("script missing %q", want)
		}
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/x-shellscript") {
		t.Errorf("content-type = %q", ct)
	}
}

func TestInstallScriptDefaults(t *testing.T) {
	s := &Server{cfg: config.Config{}} // no config at all
	rec := httptest.NewRecorder()
	s.handleInstallScript(rec, httptest.NewRequest("GET", "/install/provider.sh", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "wss://api.ayni-ai.com/ws/provider") {
		t.Errorf("missing default coordinator ws URL:\n%s", body)
	}
	if !strings.Contains(body, `MANIFEST_URL="https://models.ayni-ai.com"`) {
		t.Errorf("missing default registry URL")
	}
	for _, want := range []string{`https://api.ayni-ai.com/v1/manifest-key`, `SC_MANIFEST_VERIFY_KEY="$VERIFY_KEY"`, `SC_REQUIRE_MANIFEST="${SC_REQUIRE_MANIFEST:-1}"`} {
		if !strings.Contains(body, want) {
			t.Errorf("installer must pin the manifest key; missing %q", want)
		}
	}
}

// TestInstallScriptVerifiesBinaryAndModel: the installer must refuse an unverified
// binary and must not hand the daemon a model file it downloaded itself.
func TestInstallScriptVerifiesBinaryAndModel(t *testing.T) {
	s := &Server{cfg: config.Config{}}
	rec := httptest.NewRecorder()
	s.handleInstallScript(rec, httptest.NewRequest("GET", "/install/provider.sh", nil))
	body := rec.Body.String()
	for _, want := range []string{
		"provider-daemon-darwin-arm64) echo e2d068e6aaf8067ee1879aa4d81e4d6652831a21caeb5bb0ab73c27daea73b76 ;;",
		"provider-daemon-linux-x64) echo d3675b1bed380d6cd8b633596f5e73319608b4df649b79b6120b4c1a4e2fa997 ;;",
		"checksum mismatch for $asset: refusing to run it.",
		`SC_BUILD_FROM_SOURCE:-}" != "1"`,
		`SC_MODEL_DIR="$DIR/models"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("installer missing %q", want)
		}
	}
	for _, banned := range []string{"SC_MODEL_PATH", `curl -fSL "$MODEL_URL"`} {
		if strings.Contains(body, banned) {
			t.Errorf("installer still contains %q: the model must come through the signed registry", banned)
		}
	}
	if dump := os.Getenv("AYNI_DUMP_INSTALLER"); dump != "" {
		if err := os.WriteFile(dump, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
