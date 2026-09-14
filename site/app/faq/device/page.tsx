import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "What runs on my device — Ayni",
  description:
    "What the Ayni provider does on your phone or computer, what it costs in battery, data and heat, and how to stop it.",
};

export default function DeviceFAQ() {
  return (
    <section className="wrap" style={{ maxWidth: 760 }}>
      <p className="kicker">Plain answers</p>
      <h1>What runs on my device, and what does it cost me?</h1>
      <p className="lead">
        Short version: one process, one 491 MB model file downloaded once, jobs that take a few seconds each, and nothing
        that touches your files, contacts or browsing. Here is the longer version, measured on our own devices.
      </p>

      <h2>What gets installed</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li><strong>On a phone:</strong> the Ayni app from Google Play. It bundles the inference engine (llama.cpp) and downloads the model on first run.</li>
        <li><strong>On a Mac or Linux box:</strong> one binary, <code>provider-daemon</code>, in your home folder under <code>~/.ayni</code>, started by a one-line installer. Same engine, same model.</li>
        <li><strong>The model:</strong> Qwen 2.5 0.5B, 4-bit, one file of 491 MB, verified against a signed manifest before it is used. It is the only model in production today.</li>
      </ul>

      <h2>When it works</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>The phone app serves jobs only while the phone is charging and on Wi-Fi. Unplug it or leave Wi-Fi and it stops taking work.</li>
        <li>The desktop daemon reports its CPU load and thermal state to the coordinator with every heartbeat, and the scheduler ranks busy or hot machines last. It never takes more than a handful of jobs at once.</li>
        <li>A job is one prompt in, one completion out. In production today a job is typically a few hundred tokens and finishes in one to ten seconds on a phone.</li>
      </ul>

      <h2>Battery, data and heat</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li><strong>Battery:</strong> the phone only serves while plugged in, so the cost is a slightly slower charge during a job, not a drained battery.</li>
        <li><strong>Data:</strong> the model download is a one-time 491 MB. Jobs are small: prompt and completion text plus a few hundred bytes of framing, so a busy day is a few megabytes. Wi-Fi only on phones.</li>
        <li><strong>Heat:</strong> a job runs the CPU or GPU flat out for a few seconds. The phone reports its thermal state; a hot phone is not sent work until it cools. Sustained throughput is measured at benchmark time and capped there.</li>
        <li><strong>Storage:</strong> the model file plus a few megabytes for the app or binary.</li>
      </ul>

      <h2>What it never does</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>Read your files, photos, contacts, messages or browsing. The app requests no such permissions.</li>
        <li>Accept inbound connections. The provider dials out to the coordinator over an encrypted WebSocket and nothing can connect to your device.</li>
        <li>Show ads, track you, or phone anything but the coordinator and the model registry.</li>
        <li>Keep the prompts it serves. They live in memory for the length of the job and are never written to disk or logged; a CI check fails the build if any code path could log one.</li>
      </ul>

      <h2>How to stop it</h2>
      <ul style={{ lineHeight: 1.8 }}>
        <li>Phone: toggle sharing off in the app, or uninstall. The model file goes with the app.</li>
        <li>Mac or Linux: <code>~/.ayni/provider-daemon</code> is a normal process; stop it or delete <code>~/.ayni</code>. Nothing else was installed.</li>
      </ul>

      <p>
        Everything above is open source and checkable: <a href="https://github.com/Ayni-ai/shared-compute">the repository</a>,
        the <a href="https://github.com/Ayni-ai/shared-compute/blob/main/docs/THREAT-MODEL.md">threat model</a>, and the white paper linked from the home page.
      </p>
    </section>
  );
}
