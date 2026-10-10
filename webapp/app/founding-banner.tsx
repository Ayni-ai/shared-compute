"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type Founding } from "./lib/api";

/** One line on the dashboard: the founding-member offer, the member's progress, or their number. */
export default function FoundingBanner() {
  const [f, setF] = useState<Founding | null>(null);
  useEffect(() => { api.founding().then(setF).catch(() => setF(null)); }, []);
  if (!f) return null;
  if (!f.enrolled && !f.open) return null;

  let title: string, body: string, cta: string;
  if (f.validated) {
    title = `Founding member #${f.number}`;
    body = "Your badge is ready to share, and your link invites people in your name.";
    cta = "Share your badge";
  } else if (f.enrolled) {
    const done = [f.jobs_served > 0, f.online_minutes >= f.min_online_hours * 60, f.feedback > 0].filter(Boolean).length;
    title = `Test month: ${done} of 3 done`;
    body = "A device that served a job, a day online in total, one piece of feedback. Your number arrives when all three are in.";
    cta = "See your checklist";
  } else {
    title = `Become a founding member (${f.members.toLocaleString()} of ${f.cap.toLocaleString()} so far)`;
    body = "Thirty days of test jobs on a device you are not using, one honest piece of feedback, and a numbered badge that is yours for good.";
    cta = "Start my test month";
  }
  return (
    <section className="card" style={{ marginBottom: 18, borderColor: "#1f9e94", background: "#f0faf8" }}>
      <h3 style={{ margin: 0 }}>{title}</h3>
      <p className="muted" style={{ margin: "6px 0 10px", fontSize: ".9rem" }}>{body}</p>
      <Link href="/founding/" className="btn btn-teal">{cta}</Link>
    </section>
  );
}
