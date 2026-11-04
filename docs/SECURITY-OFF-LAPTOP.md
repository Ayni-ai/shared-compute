# Nothing on the founder's laptop (security plan, 2026-11-03)

Decision (founder, 2026-11-03): no Ayni service, secret or workload runs on a personal
machine. Everything runs in Google Cloud, in Ayni-only projects that hold no personal data.
This closes the risk found in the review the same day: the provider daemon (llama.cpp
parsing third-party text) ran in the same macOS account that held the admin token, the
production credentials and the model-registry signing key. One memory-safety bug in the
model runtime would have handed an attacker the key that every device trusts.

## Where everything lives now

| What | Before | After |
|---|---|---|
| Team provider device | MacBook Pro, founder's account (launchd) | VM `fleet-N` in project `ayni-mesh`, systemd sandbox, user `ayni` (`infra/gcp/fleet/`) |
| Tester pulse (test jobs every 10 min) | launchd on the laptop, state in `~/.ayni` | Cloud Run job `ayni-pulse` + Cloud Scheduler, state in `gs://ayni1-507216-pulse` (`infra/gcp/pulse/`) |
| Model-registry signing key | `secrets/ayni-registry-signing.key` on the laptop | Cloud KMS `ayni/registry-signer`, imported so the pinned public key is unchanged (`infra/gcp/security/registry-key-to-kms.sh`) |
| Registry publishing | `infra/publish-registry.sh` on the laptop | same script, signing through KMS; run from Cloud Shell or Cloud Build |
| Admin token, consumer key, provider token | `.env.local` on the laptop | Secret Manager only; rotated after the move; `.env.local` deleted |
| Founding report | laptop | Cloud Shell (reads `SC_ADMIN_TOKEN` from Secret Manager) |

Stopped on 2026-11-03: the laptop's `com.ayni.provider` and `com.ayni.tester-pulse` agents
(plists kept in `~/Library/LaunchAgents/disabled-ayni/` until the cloud replacements run).

## Isolation

- **Production** (`ayni1-507216`): coordinator, Cloud SQL, KMS, Secret Manager, pulse job.
  Only the founder's Ayni account and service accounts have roles.
- **Fleet** (`ayni-mesh`): provider VMs only. Own VPC, no ingress rules, egress 80/443 only,
  Cloud NAT, no external IPs, Shielded VM. The VM service account can read exactly one
  secret (the provider registration token). No role in production.
- **On each VM**: the provider runs as system user `ayni` under systemd with
  `ProtectSystem=strict`, `ProtectHome`, `NoNewPrivileges`, no capabilities, a syscall
  filter, `MemoryDenyWriteExecute`, write access to `/var/lib/ayni` only, and an iptables
  owner rule that lets it reach HTTPS and DNS but not the metadata server's credentials.
- **Mesh** stays a lab: dedicated VMs in `ayni-mesh` only, known peers only, never a
  personal machine, never a port opened on a home router.

## Supply chain

- The installer (`/install/provider.sh`) refuses any provider binary whose SHA-256 is not
  pinned in `coordinator/internal/api/provider-sha256.txt`; builds from source only with
  `SC_BUILD_FROM_SOURCE=1`. New releases change the pins in a reviewed commit.
- The installer no longer downloads the model; the daemon fetches it from the signed
  registry and checks every file's SHA-256 (`SC_MANIFEST_URL` + `SC_REGISTRY_PUBKEY`).
- The model store rejects manifest file names that are not a single plain path component,
  so even a manifest signed with a stolen key cannot write outside its model directory.
- Open: sign and notarize the macOS binary (needs an Apple Developer account, founder).

## Order of execution

1. Founder: `gcloud auth login mc@zalesgen.com` (password).
2. Import the registry key into KMS and prove the signatures match; founder deletes the seed.
3. Fleet: `create.sh setup`, store the registration token, `create.sh vm fleet-1`; confirm the
   VM registers and serves a pulse job.
4. Pulse: create the dedicated consumer key `SC_PULSE_CONSUMER_KEY`, build, `deploy.sh`.
5. Deploy the coordinator with the hardened installer.
6. Rotate everything that lived on the laptop: admin token, consumer key, provider
   registration token (new value straight into Secret Manager; the fleet secret too).
7. Founder deletes `.env.local`, `secrets/`, `~/.ayni` and the disabled plists, and revokes
   the laptop's gcloud credentials for mc@zalesgen.com if desired (`gcloud auth revoke`).
