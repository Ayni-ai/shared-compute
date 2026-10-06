import type { Metadata } from "next";
import FoundingLive from "../founding-live";

const title = "Founding members — Ayni";
const description =
  "Be one of the first 1,000. Give Ayni a test month on a device you are not using, tell us what you found, and get a numbered founding-member badge the network itself verifies.";

export const metadata: Metadata = {
  title,
  description,
  // Drawn at build time by the app (webapp/app/founding-og.png/route.tsx).
  openGraph: { title, description, url: "https://ayni-ai.com/founding/", images: [{ url: "https://app.ayni-ai.com/founding-og.png", width: 1200, height: 630 }] },
  twitter: { card: "summary_large_image", title, description, images: ["https://app.ayni-ai.com/founding-og.png"] },
};

const APP = "https://app.ayni-ai.com";
const REPO = "https://github.com/Ayni-ai/shared-compute";

export default function Founding() {
  return (
    <section className="wrap" style={{ maxWidth: 780 }}>
      <p className="kicker">Founding members · test month</p>
      <h1>Be one of the first 1,000.</h1>
      <p className="lead">
        Ayni is an open-source network that runs private AI jobs on idle phones and laptops, each one sealed to a single device so
        nobody, including us, can read it. Before we ask a single buyer to trust it, we need a thousand people to test it for a month.
        Those people are our founding members.
      </p>

      <div className="cta" style={{ margin: "24px 0 16px", display: "flex", gap: 12, flexWrap: "wrap" }}>
        <a href={`${APP}/founding/`} className="btn btn-primary">Start your test month</a>
        <a href={REPO} className="btn btn-ghost">Read the code first</a>
      </div>

      <FoundingLive />

      <h2>The deal</h2>
      <div className="grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: 16 }}>
        <div className="card">
          <h3>You get</h3>
          <ul style={{ lineHeight: 1.7, paddingLeft: 18, margin: 0 }}>
            <li>A numbered badge, #1 to #1,000, for LinkedIn, X or your README, with a page anyone can check.</li>
            <li>A direct line to the team. Every message answered, usually within a day.</li>
            <li>Credit by name in the release notes when your report ships, if you want it.</li>
          </ul>
        </div>
        <div className="card">
          <h3>You accept</h3>
          <ul style={{ lineHeight: 1.7, paddingLeft: 18, margin: 0 }}>
            <li>For 30 days your device gets test jobs a few times a day, not a steady stream of paid work.</li>
            <li>Earnings are real and metered, but cents.</li>
            <li>Things will break. Finding where is the point.</li>
          </ul>
        </div>
        <div className="card">
          <h3>The network checks</h3>
          <ul style={{ lineHeight: 1.7, paddingLeft: 18, margin: 0 }}>
            <li>One of your devices served a job.</li>
            <li>Your devices were online 24 hours in total. A phone charging overnight does a third of that.</li>
            <li>You sent one piece of feedback: a bug, a confusing screen, an idea.</li>
          </ul>
        </div>
      </div>
      <p className="muted" style={{ marginTop: 10 }}>
        Nobody hands out the badge: the coordinator checks all three every five minutes and numbers members in the order they enrolled.
        It cannot be bought, and once it is yours it stays yours.
      </p>

      <h2>What the month looks like</h2>
      <ol style={{ lineHeight: 1.8 }}>
        <li><strong>Day 0, ten minutes.</strong> Sign in at <a href={APP}>app.ayni-ai.com</a> with GitHub or Google. Add a device: an Android phone with a six-letter code, or a Mac or Linux box with one command. A test job reaches it within minutes.</li>
        <li><strong>Days 1 to 30.</strong> Leave it running when it suits you. The phone app only works while charging and on Wi-Fi; a laptop shares only while it is on. A few test jobs a day.</li>
        <li><strong>Any time.</strong> Tell us one thing from the testers page or the phone app. We answer in the same place.</li>
        <li><strong>When all three are done.</strong> Your number arrives, with a badge to download, a link that previews with your number, and an “Add to profile” button for LinkedIn.</li>
      </ol>

      <h2>Why a month without paid work</h2>
      <p>
        Buyers will only send private data to a network that is proven, and a network is only proven when enough devices are online that
        a job always finds one. Our bar is a thousand members with a hundred devices online at any moment, measured every five minutes
        and published above. Your month is what gets us there. We will say so here, with the numbers, the day it happens.
      </p>

      <h2>Who this is for</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>People who already run local models and want to see one run privately across other people’s hardware.</li>
        <li>Homelab and self-host owners with machines that sit idle by design.</li>
        <li>Anyone with a spare Android phone in a drawer, or a laptop that spends the night on its charger.</li>
        <li>Students and clubs who want their name on an open-source project from its first month.</li>
      </ul>

      <h2>What we will not do</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>No token, no blockchain, no promise of income. Earnings are dollars, metered, and small.</li>
        <li>No ads, no tracking, no access to your files. Jobs are sealed to your device with a one-time key and never logged.</li>
        <li>No lock-in: stop any time from the app or with Ctrl-C. Your badge stays.</li>
      </ul>
      <p>
        Plain answers to what everyone asks: <a href="/faq/device/">what runs on my device and what it costs me</a>,{" "}
        <a href="/faq/sealed/">how a job is sealed</a>, and <a href="/faq/earnings/">what I earn and when</a>.
      </p>

      <div className="cta" style={{ margin: "28px 0 8px", display: "flex", gap: 12, flexWrap: "wrap" }}>
        <a href={`${APP}/founding/`} className="btn btn-primary">Start your test month</a>
        <a href={`${APP}/founders/`} className="btn btn-ghost">See the founders board</a>
      </div>
      <p className="muted" style={{ marginTop: 18 }}>
        Questions first? Ask in <a href={`${REPO}/discussions`}>Discussions</a> or write to hello@ayni-ai.com.
      </p>
    </section>
  );
}
