#!/usr/bin/env bash
# Deploy the tester pulse as a Cloud Run job triggered every 10 minutes by Cloud Scheduler.
# Identity: SA ayni-pulse, which can read two secrets and one state bucket, nothing else.
#   CLOUDSDK_CORE_ACCOUNT=mc@zalesgen.com infra/gcp/pulse/deploy.sh <tag>
set -euo pipefail
cd "$(dirname "$0")/../../.."
P="${GCP_PROJECT:-ayni1-507216}"; R="${GCP_REGION:-us-east1}"
TAG="${1:-$(git rev-parse --short HEAD)}"
IMG="$R-docker.pkg.dev/$P/ayni/pulse:$TAG"
SA="ayni-pulse@$P.iam.gserviceaccount.com"
SCHED_SA="ayni-pulse-scheduler@$P.iam.gserviceaccount.com"
BUCKET="$P-pulse"

newest_enabled() {
  gcloud secrets versions list "$1" --project="$P" --filter="state=enabled" \
    --sort-by=~createTime --limit=1 --format="value(name.basename())"
}

gcloud iam service-accounts describe "$SA" --project "$P" >/dev/null 2>&1 || \
  gcloud iam service-accounts create ayni-pulse --project "$P" --display-name "Ayni tester pulse job"
gcloud iam service-accounts describe "$SCHED_SA" --project "$P" >/dev/null 2>&1 || \
  gcloud iam service-accounts create ayni-pulse-scheduler --project "$P" --display-name "Triggers the pulse job"
gcloud storage buckets describe "gs://$BUCKET" >/dev/null 2>&1 || \
  gcloud storage buckets create "gs://$BUCKET" --project "$P" --location "$R" \
    --uniform-bucket-level-access --public-access-prevention
gcloud storage buckets add-iam-policy-binding "gs://$BUCKET" \
  --member "serviceAccount:$SA" --role roles/storage.objectUser >/dev/null
for s in SC_ADMIN_TOKEN SC_PULSE_CONSUMER_KEY; do
  gcloud secrets add-iam-policy-binding "$s" --project "$P" \
    --member "serviceAccount:$SA" --role roles/secretmanager.secretAccessor >/dev/null
done

ADMIN_V="$(newest_enabled SC_ADMIN_TOKEN)"; KEY_V="$(newest_enabled SC_PULSE_CONSUMER_KEY)"
[ -n "$ADMIN_V" ] && [ -n "$KEY_V" ] || { echo "SC_ADMIN_TOKEN and SC_PULSE_CONSUMER_KEY need an enabled version"; exit 1; }

gcloud run jobs deploy ayni-pulse --project "$P" --region "$R" --image "$IMG" \
  --service-account "$SA" --tasks 1 --max-retries 0 --task-timeout 300s \
  --cpu 1 --memory 512Mi \
  --set-env-vars "PULSE_STATE_GCS=gs://$BUCKET/state.json,AYNI_BASE_URL=https://api.ayni-ai.com" \
  --set-secrets "SC_ADMIN_TOKEN=SC_ADMIN_TOKEN:$ADMIN_V,SC_CONSUMER_KEY=SC_PULSE_CONSUMER_KEY:$KEY_V"

gcloud run jobs add-iam-policy-binding ayni-pulse --project "$P" --region "$R" \
  --member "serviceAccount:$SCHED_SA" --role roles/run.invoker >/dev/null
URI="https://run.googleapis.com/v2/projects/$P/locations/$R/jobs/ayni-pulse:run"
if gcloud scheduler jobs describe ayni-pulse --location "$R" --project "$P" >/dev/null 2>&1; then
  gcloud scheduler jobs update http ayni-pulse --location "$R" --project "$P" \
    --schedule "*/10 * * * *" --uri "$URI" --http-method POST --oauth-service-account-email "$SCHED_SA"
else
  gcloud scheduler jobs create http ayni-pulse --location "$R" --project "$P" \
    --schedule "*/10 * * * *" --uri "$URI" --http-method POST --oauth-service-account-email "$SCHED_SA"
fi
echo "==> pulse job deployed ($IMG); first run: gcloud run jobs execute ayni-pulse --region $R --project $P --wait"
