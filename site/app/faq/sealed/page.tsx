import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "How a job is sealed — Ayni",
  description: "How Ayni encrypts each job to one device, what the coordinator can and cannot see, and what a device owner can see.",
};

export default function SealedFAQ() {
  return (
    <section className="wrap" style={{ maxWidth: 760 }}>
      <p className="kicker">Plain answers</p>
      <h1>How a job is sealed</h1>
      <p className="lead">
        Every job is encrypted to exactly one device with a key that is used once and thrown away. The coordinator moves
        ciphertext and counts tokens; it does not keep the text. Here is the path a prompt takes.
      </p>

      <h2>The path of one prompt</h2>
      <ol style={{ lineHeight: 1.8 }}>
        <li>A buyer sends a prompt to the coordinator over TLS, like any API call.</li>
        <li>The coordinator picks one device for the job and generates a fresh key pair for this job only (X25519).</li>
        <li>It seals the prompt to that device's registered public key (NaCl box) and forwards the sealed job. The device is the only party that can open it.</li>
        <li>The device runs the model and returns the completion sealed the same way, chunk by chunk.</li>
        <li>The coordinator decrypts each chunk only at the relay boundary, in memory, to stream it back to the buyer and count tokens. Nothing is written to disk and nothing is logged.</li>
        <li>The job key is discarded. A later compromise of the coordinator cannot decrypt past jobs.</li>
      </ol>

      <h2>Who can see what</h2>
      <table style={{ borderCollapse: "collapse", width: "100%", lineHeight: 1.6 }}>
        <thead><tr><th style={{ textAlign: "left", padding: "6px 8px" }}>Party</th><th style={{ textAlign: "left", padding: "6px 8px" }}>Sees</th></tr></thead>
        <tbody>
          <tr><td style={{ padding: "6px 8px" }}>The buyer</td><td style={{ padding: "6px 8px" }}>Their own prompt and completion, which device served it, and the price.</td></tr>
          <tr><td style={{ padding: "6px 8px" }}>The device owner</td><td style={{ padding: "6px 8px" }}>At the community tier, the owner is not prevented from inspecting the plaintext in memory on their own device. Buyers who need more require the attested tier.</td></tr>
          <tr><td style={{ padding: "6px 8px" }}>The coordinator</td><td style={{ padding: "6px 8px" }}>Ciphertext in transit, token counts, timing, model, tier and which device. Plaintext exists only transiently at the relay boundary and is never stored.</td></tr>
          <tr><td style={{ padding: "6px 8px" }}>Anyone on the network path</td><td style={{ padding: "6px 8px" }}>TLS on the outside, a per-job seal on the inside. Nothing readable.</td></tr>
        </tbody>
      </table>

      <h2>Trust tiers</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li><strong>community:</strong> signed provider binary and mandatory encryption. Today's default.</li>
        <li><strong>device_attested:</strong> Android phones whose keystore, verified boot and lock state were checked against Google's root. A buyer asks for it with one header and pays 1.4×.</li>
        <li><strong>confidential:</strong> hardware enclaves; on the roadmap, priced at 3×.</li>
      </ul>

      <h2>What keeps this honest</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>A CI check that fails the build if any code path could log a prompt.</li>
        <li>The public demo shows the entire record the network keeps for a job; search it for a sentence from your text and it is not there.</li>
        <li>Every workload's shape (counts, tier, price, never content) is reviewed by a fail-closed Council whose decisions are published with a hash chain.</li>
        <li>All of it is Apache 2.0. <a href="https://github.com/Ayni-ai/shared-compute/blob/main/docs/THREAT-MODEL.md">Read the threat model</a> and tell us what is wrong with it.</li>
      </ul>
    </section>
  );
}
