import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "What I earn and when — Ayni",
  description: "The Ayni rate card, the device owner's share, and how and when payouts happen. Cents today, and why.",
};

export default function EarningsFAQ() {
  return (
    <section className="wrap" style={{ maxWidth: 760 }}>
      <p className="kicker">Plain answers</p>
      <h1>What I earn, and when</h1>
      <p className="lead">
        Cents today. This page shows exactly how a job turns into money for your device, so you can see why, and what would
        change it.
      </p>

      <h2>The rate card</h2>
      <p>Buyers pay per token. For the model class in production today (MICRO, models up to about 3B parameters):</p>
      <table style={{ borderCollapse: "collapse", lineHeight: 1.6 }}>
        <tbody>
          <tr><td style={{ padding: "6px 12px 6px 0" }}>Input tokens</td><td style={{ padding: "6px 8px" }}>$0.02 per million</td></tr>
          <tr><td style={{ padding: "6px 12px 6px 0" }}>Output tokens</td><td style={{ padding: "6px 8px" }}>$0.08 per million</td></tr>
        </tbody>
      </table>
      <p>
        Larger classes are priced higher (SMALL $0.05 / $0.20, MEDIUM $0.15 / $0.60 per million) and will pay more per job when
        GPU devices join. Attested devices earn 1.4× the community rate; spot jobs pay 60%.
      </p>

      <h2>Your share</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li><strong>70%</strong> of what a job bills goes to the device that served it (60% on the confidential tier when it exists).</li>
        <li>Your share is credited to your account the moment the job completes, and shown on your earnings page.</li>
        <li>Ayni's part covers coordination, relay egress, the failure buffer and margin. The full build-up of a quote is itemised for the buyer, so nothing is hidden from either side.</li>
      </ul>

      <h2>Why it is cents</h2>
      <p>
        A phone serves about 18 tokens per second. A test job today is a few hundred tokens, so a job earns a fraction of a cent.
        Two things move that number: paid workloads from buyers (thousands of items, not three), and larger model classes with
        higher rates. Both are what this community is being built for. Until they arrive, your device is here for the early look
        and the influence, not the income, and we say so everywhere.
      </p>

      <h2>When you get paid</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>Payouts run weekly through Stripe Connect, to a bank account you link in a Stripe-hosted form. No token, no wallet.</li>
        <li>The minimum payout is <strong>$10</strong>. Balances below that roll over week to week; nothing expires.</li>
        <li>Sub-cent remainders carry forward too.</li>
        <li>Stripe handles identity checks and tax forms where they apply; Ayni never sees your bank details.</li>
      </ul>

      <h2>What you can check</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>Your earnings page lists jobs, devices and the amount owed.</li>
        <li>The public <a href="/status/">status page</a> shows the fleet and jobs served, so you can see how busy the network is.</li>
        <li>The rate card and the share are constants in <a href="https://github.com/Ayni-ai/shared-compute/blob/main/coordinator/internal/pricing/pricing.go">the open source code</a>, and the payment design is in <a href="https://github.com/Ayni-ai/shared-compute/blob/main/docs/PAYMENTS.md">docs/PAYMENTS.md</a>.</li>
      </ul>
    </section>
  );
}
