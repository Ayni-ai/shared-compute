#!/bin/bash
# Boot script for a team-operated provider VM (Ubuntu 24.04, project ayni-mesh).
# Runs as root on every boot; idempotent. The provider itself runs as the locked-down
# system user "ayni" under systemd, with no inbound network, egress limited to HTTPS
# and DNS, no access to the metadata server, and write access to /var/lib/ayni only.
set -euo pipefail
md() { curl -fsS -H 'Metadata-Flavor: Google' "http://metadata.google.internal/computeMetadata/v1/$1"; }

id ayni >/dev/null 2>&1 || useradd --system --home-dir /var/lib/ayni --shell /usr/sbin/nologin ayni
install -d -o ayni -g ayni -m 0700 /var/lib/ayni /var/lib/ayni/models
install -d -o root -g root -m 0755 /opt/ayni

# 1. Binary: only the release whose SHA-256 is pinned in instance metadata.
WANT="$(md instance/attributes/provider-sha256)"
BIN=/opt/ayni/provider-daemon
if [ ! -x "$BIN" ] || [ "$(sha256sum "$BIN" | cut -d' ' -f1)" != "$WANT" ]; then
  curl -fsSL --retry 3 -o "$BIN.part" \
    https://github.com/Ayni-ai/shared-compute/releases/download/provider-latest/provider-daemon-linux-x64
  if [ "$(sha256sum "$BIN.part" | cut -d' ' -f1)" != "$WANT" ]; then
    echo "ayni: provider-daemon checksum mismatch, refusing to install" >&2
    rm -f "$BIN.part"; exit 1
  fi
  install -o root -g root -m 0755 "$BIN.part" "$BIN"; rm -f "$BIN.part"
fi

# 2. Secrets and pins: read as root, handed to systemd in a root-only file.
TOKEN="$(md instance/service-accounts/default/token | python3 -c 'import sys,json;print(json.load(sys.stdin)["access_token"])')"
PROJECT="$(md project/project-id)"
REG="$(curl -fsS -H "Authorization: Bearer $TOKEN" \
  "https://secretmanager.googleapis.com/v1/projects/$PROJECT/secrets/provider-registration-token/versions/latest:access" \
  | python3 -c 'import sys,json,base64;print(base64.b64decode(json.load(sys.stdin)["payload"]["data"]).decode().strip())')"
VERIFY="$(curl -fsS https://api.ayni-ai.com/v1/manifest-key \
  | python3 -c 'import sys,json;d=json.load(sys.stdin);print(d["signer_id"]+":"+d["public_key"])')"
umask 077
cat > /etc/ayni-provider.env <<ENV
SC_COORDINATOR_URL=wss://api.ayni-ai.com/ws/provider
SC_REGISTRATION_TOKEN=$REG
SC_MODEL=qwen2.5-0.5b-instruct-q4_k_m
SC_BACKEND=llama
SC_MANIFEST_URL=https://models.ayni-ai.com
SC_REGISTRY_PUBKEY=$(md instance/attributes/registry-pubkey)
SC_MODEL_DIR=/var/lib/ayni/models
SC_IDENTITY_PATH=/var/lib/ayni/identity.key
SC_MANIFEST_VERIFY_KEY=$VERIFY
SC_REQUIRE_MANIFEST=1
ENV
unset REG TOKEN
umask 022

# 3. Per-process egress for the provider user: HTTPS and DNS only; never the metadata
#    server's HTTP API (that is where the VM's credentials live).
UID_AYNI="$(id -u ayni)"
if command -v iptables >/dev/null; then
  iptables -D OUTPUT -m owner --uid-owner "$UID_AYNI" -j AYNI 2>/dev/null || true
  iptables -F AYNI 2>/dev/null || iptables -N AYNI
  iptables -A AYNI -o lo -j ACCEPT
  iptables -A AYNI -p tcp -d 169.254.169.254 --dport 80 -j REJECT
  iptables -A AYNI -p udp --dport 53 -j ACCEPT
  iptables -A AYNI -p tcp --dport 53 -j ACCEPT
  iptables -A AYNI -p tcp --dport 443 -j ACCEPT
  iptables -A AYNI -j REJECT
  iptables -A OUTPUT -m owner --uid-owner "$UID_AYNI" -j AYNI
  ip6tables -A OUTPUT -m owner --uid-owner "$UID_AYNI" -j REJECT 2>/dev/null || true
fi

# 4. The service, sandboxed by systemd.
cat > /etc/systemd/system/ayni-provider.service <<'UNIT'
[Unit]
Description=Ayni provider (team-operated, sandboxed)
After=network-online.target
Wants=network-online.target

[Service]
User=ayni
Group=ayni
EnvironmentFile=/etc/ayni-provider.env
ExecStart=/opt/ayni/provider-daemon
Restart=always
RestartSec=10
UMask=0077
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/ayni
PrivateTmp=yes
PrivateDevices=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
ProtectClock=yes
ProtectHostname=yes
ProtectProc=invisible
ProcSubset=pid
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
RestrictNamespaces=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
LockPersonality=yes
MemoryDenyWriteExecute=yes
CapabilityBoundingSet=
AmbientCapabilities=
SystemCallArchitectures=native
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @mount @debug @reboot @swap @module @raw-io

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable ayni-provider.service
systemctl restart ayni-provider.service
echo "ayni: provider started as $(id -un ayni) with pinned binary $WANT"
