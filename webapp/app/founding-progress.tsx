"use client";
import { useEffect, useState } from "react";
import { api, type FoundingProgress } from "./lib/api";

function Bar({ value, target, label, sub }: { value: number; target: number; label: string; sub: string }) {
  const pct = Math.max(0, Math.min(100, (value / Math.max(1, target)) * 100));
  return (
    <div>
      <p style={{ margin: 0, display: "flex", justifyContent: "space-between", gap: 8 }}>
        <strong>{label}</strong>
        <span className="mono">{value.toLocaleString()} / {target.toLocaleString()}</span>
      </p>
      <div style={{ height: 10, borderRadius: 6, background: "#eef0f4", margin: "6px 0 4px", overflow: "hidden" }}>
        <div style={{ width: `${pct}%`, minWidth: value > 0 ? 4 : 0, height: "100%", background: "var(--teal)" }} />
      </div>
      <p className="muted" style={{ margin: 0, fontSize: ".82rem" }}>{sub}</p>
    </div>
  );
}

/** Public progress toward the program's two goals: 1,000 members and 100 devices online at any time. */
export default function FoundingProgressCard() {
  const [p, setP] = useState<FoundingProgress | null>(null);

  useEffect(() => {
    const load = () => api.foundingProgress().then(setP).catch(() => {});
    load();
    const t = setInterval(load, 60_000);
    return () => clearInterval(t);
  }, []);

  if (!p) return null;
  const days = p.covered_hours / 24;
  return (
    <section className="card" style={{ marginBottom: 18 }}>
      <h3>Where the founding program stands</h3>
      <div className="grid" style={{ gap: 14, marginTop: 8 }}>
        <Bar value={p.members} target={p.cap} label="Validated founding members"
          sub={`${p.enrolled.toLocaleString()} ${p.enrolled === 1 ? "person is" : "people are"} in the test month.`} />
        <Bar value={p.online_floor_7d} target={p.online_target} label="Devices online at any time"
          sub={p.covered_hours > 0
            ? `At least this many were online 95% of the time over the last ${days >= 7 ? "7 days" : days >= 1 ? `${days.toFixed(1)} days` : `${p.covered_hours} hours`}. Right now: ${p.online_now}. Peak: ${p.online_peak_7d}.`
            : `Right now: ${p.online_now}. Measured every five minutes.`} />
      </div>
    </section>
  );
}
