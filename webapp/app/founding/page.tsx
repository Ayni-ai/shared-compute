"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type Founding } from "../lib/api";
import FoundingProgressCard from "../founding-progress";

function Step({ done, title, children }: { done: boolean; title: string; children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", gap: 12, padding: "12px 0", borderTop: "1px solid var(--line)" }}>
      <span aria-hidden style={{ fontSize: "1.2rem", lineHeight: 1.2, color: done ? "#0f6b63" : "#9aa3b5" }}>{done ? "✓" : "○"}</span>
      <div style={{ flex: 1 }}>
        <p style={{ margin: 0, fontWeight: 600 }}>
          {title} {done && <span className="pill" style={{ marginLeft: 6 }}>done</span>}
        </p>
        <div className="muted" style={{ margin: "4px 0 0", fontSize: ".9rem" }}>{children}</div>
      </div>
    </div>
  );
}

// Renders the server's SVG card to a PNG the member can post anywhere. The SVG is
// fetched as text and drawn from a blob URL, so the canvas is never cross-origin.
async function downloadPNG(svgURL: string, filename: string) {
  const res = await fetch(svgURL);
  if (!res.ok) throw new Error(`badge HTTP ${res.status}`);
  const blobURL = URL.createObjectURL(new Blob([await res.text()], { type: "image/svg+xml" }));
  try {
    const img = new Image();
    await new Promise((ok, fail) => { img.onload = ok; img.onerror = fail; img.src = blobURL; });
    const c = document.createElement("canvas");
    c.width = 2400; c.height = 1260;
    c.getContext("2d")!.drawImage(img, 0, 0, c.width, c.height);
    const png = await new Promise<Blob | null>((r) => c.toBlob(r, "image/png"));
    if (!png) throw new Error("could not render");
    const a = document.createElement("a");
    a.href = URL.createObjectURL(png);
    a.download = filename;
    a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  } finally {
    URL.revokeObjectURL(blobURL);
  }
}

function fmtDate(s?: string | null) {
  return s ? new Date(s).toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" }) : "";
}

export default function FoundingPage() {
  const [f, setF] = useState<Founding | null>(null);
  const [signedIn, setSignedIn] = useState<boolean | null>(null);
  const [accept, setAccept] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [copied, setCopied] = useState("");

  const load = () =>
    api.founding().then((x) => { setF(x); setSignedIn(true); }).catch((e: any) => {
      if (e?.status === 401) setSignedIn(false);
      else setErr(e?.message || "could not load");
    });

  useEffect(() => {
    load();
    const t = setInterval(load, 60_000);
    return () => clearInterval(t);
  }, []);

  const enroll = async () => {
    setBusy(true); setErr("");
    try { setF(await api.enrollFounding()); } catch (e: any) { setErr(e.message || "could not enroll"); } finally { setBusy(false); }
  };
  const copy = async (label: string, text: string) => {
    try { await navigator.clipboard.writeText(text); setCopied(label); setTimeout(() => setCopied(""), 1500); } catch {}
  };

  const hours = f ? f.online_minutes / 60 : 0;
  const need = f?.min_online_hours ?? 24;

  return (
    <>
      <Link href={signedIn ? "/dashboard/" : "/"} className="muted">← {signedIn ? "Dashboard" : "Home"}</Link>
      <h1 style={{ marginTop: 12 }}>Founding member</h1>
      <p className="muted" style={{ margin: "6px 0 18px" }}>
        Ayni is in its test month. The first {(f?.cap ?? 1000).toLocaleString()} people who join now, keep a device online
        and tell us what they found become founding members: numbered, verifiable and permanent.
      </p>

      <section className="card" style={{ marginBottom: 18 }}>
        <h3>The deal, plainly</h3>
        <div className="grid c3" style={{ marginTop: 8 }}>
          <div>
            <p style={{ margin: 0, fontWeight: 600 }}>You get</p>
            <ul className="muted" style={{ margin: "6px 0 0 18px", padding: 0, fontSize: ".9rem", lineHeight: 1.6 }}>
              <li>A numbered founding-member badge, #1 to #1,000, that anyone can verify</li>
              <li>A team that reads and answers your feedback, and credits you when it ships</li>
              <li>Your name on the founders board, if you want it there</li>
            </ul>
          </div>
          <div>
            <p style={{ margin: 0, fontWeight: 600 }}>You accept</p>
            <ul className="muted" style={{ margin: "6px 0 0 18px", padding: 0, fontSize: ".9rem", lineHeight: 1.6 }}>
              <li>For 30 days your device gets test jobs a few times a day, not regular paid work</li>
              <li>Earnings are real but cents</li>
              <li>Things will break; that is what the month is for</li>
            </ul>
          </div>
          <div>
            <p style={{ margin: 0, fontWeight: 600 }}>We check, automatically</p>
            <ul className="muted" style={{ margin: "6px 0 0 18px", padding: 0, fontSize: ".9rem", lineHeight: 1.6 }}>
              <li>One of your devices served a job</li>
              <li>Your devices were online {need} hours in total, spread however suits you</li>
              <li>You sent us one piece of feedback</li>
            </ul>
          </div>
        </div>
      </section>

      {err && <p className="err">{err}</p>}

      {signedIn === false && (
        <section className="card" style={{ marginBottom: 18 }}>
          <p style={{ margin: 0 }}>Sign in with GitHub or Google to enroll. No password, no card.</p>
          <Link href="/" className="btn btn-primary" style={{ marginTop: 12 }}>Sign in</Link>
        </section>
      )}

      {f && !f.enrolled && (
        <section className="card" style={{ marginBottom: 18 }}>
          {f.open ? (
            <>
              <label style={{ display: "flex", gap: 10, alignItems: "flex-start", fontSize: ".95rem" }}>
                <input type="checkbox" checked={accept} onChange={(e) => setAccept(e.target.checked)} style={{ marginTop: 4 }} />
                <span>
                  I understand that during my test month my devices get test jobs, not regular paid workloads, that earnings
                  are cents, and that Ayni may change while I test it.
                </span>
              </label>
              <button className="btn btn-teal" style={{ marginTop: 12 }} disabled={!accept || busy} onClick={enroll}>
                {busy ? "Enrolling…" : "Start my test month"}
              </button>
            </>
          ) : (
            <p style={{ margin: 0 }}>All {f.cap.toLocaleString()} founding places are taken. You can still join Ayni and share a device.</p>
          )}
        </section>
      )}

      {f && f.enrolled && !f.validated && (
        <section className="card" style={{ marginBottom: 18 }}>
          <h3>Your test month</h3>
          <p className="muted" style={{ margin: "4px 0 6px", fontSize: ".9rem" }}>
            Started {fmtDate(f.enrolled_at)}{f.test_month_ends ? `, ends ${fmtDate(f.test_month_ends)}` : ""}. We check every five
            minutes; your number arrives the moment all three are done.
          </p>
          <Step done={f.jobs_served > 0} title="A device served a job">
            {f.devices === 0 ? (
              <>No device yet. <Link href="/share/">Add one</Link>: a phone with a six-letter code, or a Mac or Linux box with one command.</>
            ) : f.jobs_served > 0 ? (
              <>{f.jobs_served} {f.jobs_served === 1 ? "job" : "jobs"} served by your {f.devices === 1 ? "device" : `${f.devices} devices`}.</>
            ) : (
              <>Your device is linked. A test job reaches new devices within minutes of them coming online; leave it running.</>
            )}
          </Step>
          <Step done={hours >= need} title={`${need} hours online`}>
            <div style={{ height: 8, borderRadius: 6, background: "#eef0f4", margin: "4px 0 6px", overflow: "hidden", maxWidth: 360 }}>
              <div style={{ width: `${Math.min(100, (hours / need) * 100)}%`, height: "100%", background: "var(--teal)" }} />
            </div>
            {hours.toFixed(1)} of {need} hours, all devices together. A phone on its charger overnight does about eight.
          </Step>
          <Step done={f.feedback > 0} title="Tell us one thing">
            {f.feedback > 0 ? (
              <>Thank you. Replies arrive on the <Link href="/testers/">testers page</Link>.</>
            ) : (
              <>A bug, a confusing screen, an idea. <Link href="/testers/">Send it here</Link>; the team answers within a day.</>
            )}
          </Step>
        </section>
      )}

      {f && f.validated && (
        <section className="card" style={{ marginBottom: 18 }}>
          <h3>Founding member #{f.number}</h3>
          <p className="muted" style={{ margin: "4px 0 12px", fontSize: ".9rem" }}>
            Validated {fmtDate(f.validated_at)}. Your link shows a verified badge and invites people in your name.
          </p>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={f.card_url} alt={`Ayni founding member #${f.number}`} style={{ width: "100%", borderRadius: 12, border: "1px solid var(--line)" }} />
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginTop: 12 }}>
            <button className="btn btn-primary" onClick={() => downloadPNG(f.card_url, `ayni-founding-member-${f.number}.png`).catch((e) => setErr(e.message))}>
              Download image
            </button>
            <a className="btn btn-ghost" target="_blank" rel="noreferrer"
              href={`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(f.share_url)}`}>Share on LinkedIn</a>
            <a className="btn btn-ghost" target="_blank" rel="noreferrer"
              href={`https://x.com/intent/post?text=${encodeURIComponent(`I'm founding member #${f.number} of Ayni, the open-source network that runs private AI on idle phones and laptops. One of the first ${f.cap.toLocaleString()}.`)}&url=${encodeURIComponent(f.share_url)}`}>Share on X</a>
            <a className="btn btn-ghost" target="_blank" rel="noreferrer"
              href={"https://www.linkedin.com/profile/add?" + new URLSearchParams({
                startTask: "CERTIFICATION_NAME",
                name: "Ayni Founding Member",
                organizationName: "Ayni",
                issueYear: String(new Date(f.validated_at || Date.now()).getUTCFullYear()),
                issueMonth: String(new Date(f.validated_at || Date.now()).getUTCMonth() + 1),
                certUrl: f.member_page,
                certId: `AYNI-FM-${String(f.number).padStart(4, "0")}`,
              }).toString()}>Add to LinkedIn profile</a>
            <button className="btn btn-ghost" onClick={() => copy("link", f.share_url)}>{copied === "link" ? "Copied" : "Copy link"}</button>
          </div>
          <p style={{ margin: "16px 0 6px", fontWeight: 600 }}>For a README or a profile page</p>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={f.badge_url} alt={`Ayni founding member #${f.number}`} />
          <pre className="cmd" style={{ overflowWrap: "anywhere", whiteSpace: "pre-wrap", marginTop: 8 }}>
            {`[![Ayni founding member #${f.number}](${f.badge_url})](${f.share_url})`}
          </pre>
          <button className="btn btn-ghost" onClick={() => copy("md", `[![Ayni founding member #${f.number}](${f.badge_url})](${f.share_url})`)}>
            {copied === "md" ? "Copied" : "Copy Markdown"}
          </button>
          <p className="muted" style={{ margin: "12px 0 0", fontSize: ".85rem" }}>
            Your name appears on the badge only if you chose to show it on the founders board (Dashboard → Invite a device).
            Your feedback keeps counting: tell us what you see, and we credit founding members in every release note.
          </p>
        </section>
      )}

      <FoundingProgressCard />
    </>
  );
}
