"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { api, API, type Member } from "../lib/api";
import { REF_KEY } from "../invite-card";

/** Public verification page: app.ayni-ai.com/member/?c=<code>. */
export default function MemberView() {
  const [code, setCode] = useState("");
  const [m, setM] = useState<Member | null>(null);
  const [state, setState] = useState<"loading" | "ok" | "missing">("loading");

  useEffect(() => {
    const c = (new URLSearchParams(window.location.search).get("c") || "").toLowerCase();
    setCode(c);
    if (!c) { setState("missing"); return; }
    // The member's link is also their invite: keep the code so it applies at sign-in.
    try { if (!localStorage.getItem(REF_KEY)) localStorage.setItem(REF_KEY, c); } catch {}
    api.member(c).then((x) => { setM(x); setState("ok"); }).catch(() => setState("missing"));
  }, []);

  if (state === "loading") return <p className="muted">Checking…</p>;
  if (state === "missing" || !m) {
    return (
      <>
        <h1 style={{ marginTop: 12 }}>Not a founding member yet</h1>
        <p className="muted">This link does not belong to a validated founding member. Members are validated by the network itself: a device that
          served a job, a day of time online and one piece of feedback.</p>
        <Link href="/founders/" className="btn btn-ghost">See the founders board</Link>
      </>
    );
  }
  const when = new Date(m.validated_at).toLocaleDateString(undefined, { year: "numeric", month: "long" });
  return (
    <>
      <p className="pill" style={{ display: "inline-block" }}>✓ Verified by the Ayni network</p>
      <h1 style={{ marginTop: 12 }}>{m.name ? `${m.name} is founding member #${m.number}` : `Founding member #${m.number}`}</h1>
      <p className="muted" style={{ margin: "6px 0 18px" }}>
        One of the first {m.cap.toLocaleString()} people whose devices power Ayni, the open-source network that runs private AI jobs on
        idle phones and laptops, encrypted so nobody, including us, can read them. Validated {when}.
      </p>
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img src={`${API}/v1/members/${encodeURIComponent(code)}/badge.svg?style=card`} alt={`Ayni founding member #${m.number}`}
        style={{ width: "100%", borderRadius: 12, border: "1px solid var(--line)", marginBottom: 18 }} />
      <section className="card">
        <h3>Become a founding member too</h3>
        <p className="muted" style={{ fontSize: ".92rem" }}>
          Thirty days, a phone on its charger or a laptop you are not using, and one honest piece of feedback. Your device gets test jobs, not
          a paycheck, while we get the network ready for real workloads. Places are numbered and stop at {m.cap.toLocaleString()}.
        </p>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <Link href={`/join/?c=${encodeURIComponent(code)}`} className="btn btn-primary">Join through this invite</Link>
          <Link href="/founders/" className="btn btn-ghost">Founders board</Link>
        </div>
      </section>
    </>
  );
}
