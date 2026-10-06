-- Founding members (docs/FOUNDING-MEMBERS.md). A person enrolls in the test month,
-- accepting that their devices get test jobs rather than regular paid workloads. The
-- coordinator validates them from what it observes (a device served a job, the
-- account's devices were online long enough, the person sent feedback) and gives
-- each validated member a permanent number, 1 to 1,000, in validation order.

ALTER TABLE users ADD COLUMN IF NOT EXISTS founding_enrolled_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS founding_validated_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS founding_number INTEGER;
CREATE UNIQUE INDEX IF NOT EXISTS users_founding_number_idx
    ON users (founding_number) WHERE founding_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS users_founding_pending_idx
    ON users (founding_enrolled_at) WHERE founding_enrolled_at IS NOT NULL AND founding_number IS NULL;

-- Minutes each device has been seen connected, added by the five-minute fleet sampler.
ALTER TABLE provider_owners ADD COLUMN IF NOT EXISTS online_minutes INTEGER NOT NULL DEFAULT 0;

-- Fleet size over time, one row per sample. Counts only: no device, user or job ids.
-- "100 online at any given time" is read from here as the 5th percentile of the
-- trailing seven days (online 95% of the time), not from a single good moment.
CREATE TABLE IF NOT EXISTS fleet_samples (
    id     BIGSERIAL PRIMARY KEY,
    at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    online INTEGER NOT NULL,
    phones INTEGER NOT NULL DEFAULT 0,
    pcs    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS fleet_samples_at_idx ON fleet_samples (at DESC);
