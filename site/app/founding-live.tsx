"use client";

import { useEffect, useState } from "react";

const API = "https://api.ayni-ai.com";

type Progress = {
  members: number;
  enrolled: number;
  cap: number;
  online_now: number;
  online_target: number;
  online_floor_7d: number;
  online_peak_7d: number;
  covered_hours: number;
};

/** Live progress toward the founding program's two goals. Counts only. */
export default function FoundingLive() {
  const [p, setP] = useState<Progress | null>(null);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const r = await fetch(`${API}/v1/founding`, { cache: "no-store" });
        if (r.ok && alive) setP(await r.json());
      } catch {}
    };
    load();
    const t = setInterval(load, 60_000);
    return () => {
      alive = false;
      clearInterval(t);
    };
  }, []);

  const bar = (value: number, target: number, label: string, sub: string) => (
    <div style={{ flex: "1 1 260px" }}>
      <div style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
        <strong>{label}</strong>
        <span className="mono">{value.toLocaleString()} / {target.toLocaleString()}</span>
      </div>
      <div style={{ height: 10, borderRadius: 6, background: "rgba(127,127,127,.18)", margin: "6px 0 4px", overflow: "hidden" }}>
        <div style={{ width: `${Math.min(100, (value / Math.max(1, target)) * 100)}%`, minWidth: value > 0 ? 4 : 0, height: "100%", background: "#1f9e94" }} />
      </div>
      <div style={{ fontSize: 13, opacity: 0.7 }}>{sub}</div>
    </div>
  );

  return (
    <div aria-live="polite" style={{ display: "flex", gap: 28, flexWrap: "wrap", padding: "18px 22px", border: "1px solid var(--line, #e6e2d8)", borderRadius: 12, margin: "8px 0 20px" }}>
      {p ? (
        <>
          {bar(p.members, p.cap, "Founding members", `${p.enrolled.toLocaleString()} in their test month right now.`)}
          {bar(p.online_floor_7d, p.online_target, "Devices online at any time",
            p.covered_hours > 0 ? `Online 95% of the time this week. Now: ${p.online_now}. Peak: ${p.online_peak_7d}.` : `Now: ${p.online_now}. Measured every five minutes.`)}
        </>
      ) : (
        <div style={{ fontSize: 13, opacity: 0.7 }}>Loading live numbers…</div>
      )}
    </div>
  );
}
