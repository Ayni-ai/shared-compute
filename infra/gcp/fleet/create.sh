#!/usr/bin/env bash
# Team-operated provider VMs, isolated from production (2026-11).
#
# Project ayni-mesh holds nothing but these VMs: no personal data, no production
# credentials, its own network. A compromised provider VM can read one secret (the
# provider registration token) and reach the internet on 80/443; nothing else.
#
#   CLOUDSDK_CORE_ACCOUNT=mc@zalesgen.com infra/gcp/fleet/create.sh setup          # network, SA, secret
#   ... put the registration token in the secret (printed command) ...
#   CLOUDSDK_CORE_ACCOUNT=mc@zalesgen.com infra/gcp/fleet/create.sh vm fleet-1     # one provider VM
set -euo pipefail
cd "$(dirname "$0")/../../.."
P="${FLEET_PROJECT:-ayni-mesh}"; R=us-east1; Z="${FLEET_ZONE:-us-east1-b}"
TYPE="${FLEET_MACHINE:-e2-medium}"   # 2 shared vCPU, 4 GB: enough for the 0.5B model
NET=ayni-fleet; SUB=ayni-fleet-us-east1; SA=fleet-provider; SECRET=provider-registration-token
SA_EMAIL="$SA@$P.iam.gserviceaccount.com"

setup() {
  gcloud services enable compute.googleapis.com secretmanager.googleapis.com --project "$P"
  gcloud compute networks describe "$NET" --project "$P" >/dev/null 2>&1 || \
    gcloud compute networks create "$NET" --project "$P" --subnet-mode custom
  gcloud compute networks subnets describe "$SUB" --region "$R" --project "$P" >/dev/null 2>&1 || \
    gcloud compute networks subnets create "$SUB" --project "$P" --network "$NET" --region "$R" \
      --range 10.80.0.0/24 --enable-private-ip-google-access
  gcloud compute routers describe ayni-fleet-router --region "$R" --project "$P" >/dev/null 2>&1 || \
    gcloud compute routers create ayni-fleet-router --project "$P" --network "$NET" --region "$R"
  gcloud compute routers nats describe ayni-fleet-nat --router ayni-fleet-router --region "$R" --project "$P" >/dev/null 2>&1 || \
    gcloud compute routers nats create ayni-fleet-nat --project "$P" --router ayni-fleet-router --region "$R" \
      --auto-allocate-nat-external-ips --nat-all-subnet-ip-ranges
  # No ingress rules at all (the implied rule denies). Egress: HTTP(S) only.
  gcloud compute firewall-rules describe ayni-fleet-egress-web --project "$P" >/dev/null 2>&1 || \
    gcloud compute firewall-rules create ayni-fleet-egress-web --project "$P" --network "$NET" \
      --direction EGRESS --action ALLOW --rules tcp:80,tcp:443 --destination-ranges 0.0.0.0/0 --priority 1000
  gcloud compute firewall-rules describe ayni-fleet-egress-deny --project "$P" >/dev/null 2>&1 || \
    gcloud compute firewall-rules create ayni-fleet-egress-deny --project "$P" --network "$NET" \
      --direction EGRESS --action DENY --rules all --destination-ranges 0.0.0.0/0 --priority 65000
  gcloud iam service-accounts describe "$SA_EMAIL" --project "$P" >/dev/null 2>&1 || \
    gcloud iam service-accounts create "$SA" --project "$P" --display-name "Ayni team provider VMs"
  gcloud secrets describe "$SECRET" --project "$P" >/dev/null 2>&1 || \
    gcloud secrets create "$SECRET" --project "$P" --replication-policy automatic
  gcloud secrets add-iam-policy-binding "$SECRET" --project "$P" \
    --member "serviceAccount:$SA_EMAIL" --role roles/secretmanager.secretAccessor >/dev/null
  echo
  echo "Next, store the provider registration token without printing it, e.g. from Cloud Shell:"
  echo "  read -rs T && printf %s \"\$T\" | gcloud secrets versions add $SECRET --project $P --data-file=- && unset T"
}

vm() {
  local name="$1"
  local sha pub
  sha="$(awk '$2=="provider-daemon-linux-x64"{print $1}' coordinator/internal/api/provider-sha256.txt)"
  pub="$(tr -d '[:space:]' < model-registry/PUBKEY)"
  [ -n "$sha" ] || { echo "no pinned linux-x64 hash"; exit 1; }
  gcloud compute instances create "$name" --project "$P" --zone "$Z" \
    --machine-type "$TYPE" --subnet "$SUB" --no-address \
    --image-family ubuntu-2404-lts-amd64 --image-project ubuntu-os-cloud \
    --boot-disk-size 20GB --boot-disk-type pd-balanced \
    --shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring \
    --service-account "$SA_EMAIL" --scopes cloud-platform \
    --labels role=ayni-provider,operator=team \
    --metadata "provider-sha256=$sha,registry-pubkey=$pub,block-project-ssh-keys=true,enable-oslogin=true" \
    --metadata-from-file startup-script=infra/gcp/fleet/startup.sh
}

case "${1:-}" in
  setup) setup ;;
  vm) vm "${2:?vm name}" ;;
  *) echo "usage: $0 setup | vm <name>"; exit 2 ;;
esac
