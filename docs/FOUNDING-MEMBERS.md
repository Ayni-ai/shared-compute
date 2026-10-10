# Founding members: 1,000 members, 100 devices online at any time

Written 2026-10-06. Plan of record for recruiting Ayni's first community. It narrows
[GTM-COMMUNITY-PLAN.md](GTM-COMMUNITY-PLAN.md) (1,000 devices online at once, months away) to
a goal we can reach in one quarter and measure every five minutes:

1. **1,000 validated founding members**, and
2. **100 devices online at any time**: the fleet floor, the 5th percentile of five-minute
   samples over the trailing seven days (the fleet was at least this large 95% of the time).

Both numbers are public and live: `GET /v1/founding`, [ayni-ai.com/founding](https://ayni-ai.com/founding/),
[app.ayni-ai.com/founders](https://app.ayni-ai.com/founders/).

## Who we are looking for

People who will give Ayni a month **without regular workloads**: their device gets test jobs a
few times a day (the [pulse](../scripts/tester-pulse.py)), not a paycheck. That rules out anyone
here for income, so we recruit for the motives that survive a month of cents:

| Who | Why they tolerate a test month | Where |
|---|---|---|
| People who already run local models | Curiosity about private inference across other people's hardware; they read code | r/LocalLLaMA, llama.cpp and Ollama Discords, HN |
| Homelab and self-host owners | Machines idle by design; like a binary that dials out and accepts nothing inbound | r/selfhosted, r/homelab, homelab Discords |
| Owners of a spare Android phone | It sits in a drawer; charging-and-Wi-Fi-only policy makes it free to try | r/androidapps, r/Android, XDA (after Play open testing) |
| Students and clubs | A verifiable credential from an open-source project's first month; many devices per contact | CS societies, hackathons, university Discords |
| Privacy and open-source people | The mission; a say in a product while it is still soft | HN, Mastodon, LinkedIn, privacy communities |
| The founder's network | Personal trust; highest conversion of any channel | Direct messages |

What every one of them gets is the same: **status** (a numbered badge, #1 to #1,000, that the
network itself verifies), **influence** (every message answered, shipped fixes credited by name)
and **a front-row seat**. What they give is a month of a device they were not using and one
honest piece of feedback.

## The offer and how validation works

Built in this change; the rules live in code, not in anyone's judgement.

- **Enroll** at [app.ayni-ai.com/founding](https://app.ayni-ai.com/founding/): the person ticks
  that they understand the test month (test jobs, not regular paid workloads; earnings are
  cents; things will change) → `POST /v1/me/founding/enroll`, stored as `users.founding_enrolled_at`.
- **Validate**, checked by the coordinator every five minutes (`RunFoundingLoop`,
  `auth.ValidateFoundingMembers`), all three required:
  1. one of their devices served a job (a row in `provider_earnings`, which the pulse creates
     within minutes of a device joining);
  2. their devices were online **24 hours in total** (`provider_owners.online_minutes`, credited
     by the five-minute sampler; `SC_FOUNDING_MIN_ONLINE_HOURS`);
  3. they sent **one piece of feedback** (bug, feedback or idea; access requests do not count).
- **Number**: validated members are numbered in enrollment order, 1 to 1,000, permanently.
  The program closes at 1,000.
- **Badge**: `GET /v1/members/{code}/badge.svg` (README style) and `?style=card` (1200×630 social
  card); the app turns the card into a PNG, shares to LinkedIn and X, and pre-fills LinkedIn's
  "Add license or certification" with a verification URL. The share link `api.ayni-ai.com/m/{code}`
  previews with the member's number and forwards to the public verification page
  `app.ayni-ai.com/member/?c={code}`, which is also their invite link.
- **Privacy**: the public page shows the number and the month only; the display name appears
  only if the member opted in on the founders board. Fleet samples are counts only.
- **Team accounts do not enroll.** The count is the community, not us.

## The arithmetic

### 1,000 members → 100 online at any time

Online at any time is a floor, so the fleet's average has to sit well above it:

- A mostly single-region fleet peaks at night (phones on chargers) and dips in the day; the floor
  is about **0.6 × the average** (0.75 once members spread across time zones). So 100 at the floor
  needs about **170 online on average**.
- A device that stays installed is online about 40% of the day (a phone on its charger about a
  third, a computer about half; GTM-COMMUNITY-PLAN measurements), and a member brings about
  1.3 devices.
- Average online = members × *still running after their month* × 1.3 × 0.4 = 1,000 × r × 0.52.
  For 170 we need **r ≈ 1/3**.

**So 1,000 members gives 100 online at any time if a third of them keep a device running after
their test month.** That retention number, day-60 active members, is the one to watch once
members exist; the report flags it (floor below 0.07 per member).

### Sign-ups → 1,000 validated

| Step | Target rate | Needed for 1,000 validated |
|---|---|---|
| Landing-page visit → sign-in | 12% (technical audience; personal messages and invites 30 to 50%) | ~30,000 visits |
| Sign-in → enroll | 70% (the founding offer is the main call to action) | ~3,600 sign-ins |
| Enroll → a device served a job | 70% | ~2,500 enrolled |
| → 24 h online in total | 75% | |
| → sent feedback | 80% (in-app prompt plus a day-3 nudge) | |
| Enroll → validated | **~40%** | **~2,500 enrollments** |

Invites cut the top of the funnel: at 0.25 validated invitees per validated member (the member
page is the invite), a fifth of the 1,000 arrive through members. Every rate above is an
assumption until week 2, when the report replaces it with what we measured.

## Milestones (14 weeks from launch)

Validated lags enrolled by one to two weeks (a day online, then feedback).

| Week | Enrolled | Validated | Floor | Main moves |
|---|---|---|---|---|
| 1 | 60 | 10 | 5 | 50 personal messages (draft F6), Discussions announcement (F11), team devices disclosed |
| 2 | 150 | 40 | 10 | r/LocalLLaMA (F1), llama.cpp and Ollama Discords (F8) |
| 3 | 300 | 90 | 15 | Show HN (F2), r/selfhosted (F3) |
| 4 | 450 | 150 | 25 | r/homelab (F3), first "You said, we did" note, LinkedIn badge wave (F5, F9) |
| 6 | 800 | 280 | 40 | Android communities (F4) once Play open testing is live; Product Hunt |
| 8 | 1,200 | 450 | 55 | University clubs and hackathons (F7) |
| 10 | 1,700 | 650 | 70 | Invite push: members' links, member spotlights in the weekly note |
| 12 | 2,100 | 830 | 85 | Second wave on the two channels that converted best |
| 14 | 2,500 | 1,000 | 100 | Program closes; publish the numbers |

## The weekly loop (run until both goals are met)

**Monday.** `python3 scripts/founding-report.py --record`, paste `--row` into the progress log
below, read the rules it triggered. Pick the week's channels from the milestone table, adjusted
by what converted (ask "where did you hear about us?" in the first reply to every new member and
note it beside their feedback thread). Draft the posts and messages from the kit below.

**Tuesday to Thursday.** The founder approves each post; it goes out from the founder's own
accounts (never accounts we do not own, never the same text twice on one day). Every thread is
watched for two hours and every comment answered from the reply kit.

**Every day.** Answer every feedback thread within a day from the console's Feedback page;
label it; turn real bugs into issues. New devices get a pulse job within minutes (already running).

**Friday.** "You said, we did" in Discussions → Announcements: what shipped from member feedback,
credited by display name (opt-in), plus the two numbers. Reply "shipped in <version>" on each
original thread.

### Stop/go rules

The report checks these automatically.

1. **Validation stalls**: if fewer than 25% of members enrolled a week or more ago are validated
   (once 50 have had a week), stop posting and fix the step where they stall: no device →
   onboarding; device but no job → pulse; job but < 24 h → tell people a phone on its charger
   overnight does a third; no feedback → the in-app prompt and the day-3 nudge (F10).
2. **Feedback is the bottleneck**: under 60% of members with a served job have written to us.
3. **Retention**: once 100 are validated, a floor under 0.07 per member means members leave after
   their badge. Fix the reason to stay (more pulse jobs, the founding-provider pool from
   GTM-COMMUNITY-PLAN, first design-partner workloads) before recruiting harder.
4. **A channel that yields fewer than 10 enrollments per post** gets a different message, not a
   second post.

## Launch checklist

1. Merge this change. The coordinator applies migration `0013_founding_members` on start; Pages
   rebuilds the site and the app.
2. Check: `curl https://api.ayni-ai.com/v1/founding` answers with `cap: 1000, online_target: 100`;
   ayni-ai.com/founding renders; app.ayni-ai.com/founding shows the enroll box when signed in.
3. Enroll one non-team test account end to end: device, pulse job, 24 h (or temporarily
   `SC_FOUNDING_MIN_ONLINE_HOURS=1` on a staging coordinator), feedback, badge, share preview
   (paste the `/m/` link in LinkedIn's Post Inspector and X's composer).
4. Confirm Play open testing before any phone-community post (internal testing caps at 100).
5. Start the Monday report and post week 1.

## What the agent does, and what only the founder can do

The agent built the program (enrollment, validation, badges, the landing page, the report), keeps
this plan and the drafts current, runs the weekly report when it can reach the API, drafts every
post and reply, and answers feedback threads through the console. It does not create accounts,
type credentials or post from personal accounts. The founder logs in, approves and sends posts and
personal messages, finishes Play open testing, and deploys.

## Outreach kit (founding framing)

Tone: honest about size (one small model, a small fleet, cents), specific about the deal,
no hype. Every link points to ayni-ai.com/founding or the founder's invite link.

### F1. r/LocalLLaMA

**Title:** I'm looking for 1,000 founding members for an open-source private-inference network (a month of test jobs on an idle device, a verifiable badge, your feedback shapes it)

I built Ayni, an open-source (Apache 2.0) network that runs small LLM jobs on idle phones and laptops. A Go coordinator seals each job to one device's key (X25519 + XSalsa20-Poly1305, fresh keypair per job), the device runs llama.cpp, and only ciphertext crosses the network. A CI check fails the build if any code path could log a prompt.

Before I ask anyone to pay for it, I need it tested by people who know this stack. So: founding members. For 30 days your device gets test jobs a few times a day, not paid work; earnings are real but cents. The coordinator validates you itself (a device served a job, 24 hours online in total, one piece of feedback) and gives you a number, #1 to #1,000, with a badge anyone can verify.

Honest status: one 0.5B model in production, a 7B class measured on a team Mac, a small fleet. Phone ~18 tok/s, M-series Mac 80 to 160.

Join (ten minutes, one command or a six-letter pairing code): ayni-ai.com/founding · Code and threat model: github.com/Ayni-ai/shared-compute

### F2. Show HN

**Title:** Show HN: Ayni – private LLM inference on idle devices; looking for 1,000 founding testers

Ayni turns idle phones and laptops into a small inference network. The coordinator quotes a batch up front, seals each item to one device's X25519 key, relays ciphertext, verifies token counts itself and pays the device owner in dollars through Stripe. No token, no chain.

I'm recruiting the first 1,000 people to run it for a month. Their devices get test jobs, not paid workloads, while we prove the network; the coordinator validates each member from what it observes and numbers them. The bar we publish live: 1,000 members and 100 devices online 95% of the time.

I would most like people to read the threat model and tell me what is wrong with it. https://ayni-ai.com/founding · https://github.com/Ayni-ai/shared-compute

### F3. r/selfhosted, r/homelab

**Title:** Got a box that idles? Founding members wanted for an open-source inference network (one command, dials out only, no crypto)

Ayni's provider is one binary (Rust, llama.cpp inside). One command installs it; it dials out over WebSocket, accepts nothing inbound and serves sealed jobs it cannot log. For the first month your machine gets test jobs a few times a day while we prove the network; earnings are cents. Keep it online 24 hours in total, tell us one thing that broke, and the coordinator makes you a numbered founding member (#1 to #1,000) with a badge for your README. ayni-ai.com/founding

### F4. r/androidapps

**Title:** [Founding testers] Put the spare phone in your drawer to work: open-source app runs private AI jobs only while charging on Wi-Fi

Ayni runs small AI jobs on your phone only while it is charging and on Wi-Fi, encrypted with a one-time key; no ads, no tracking, no access to your files. It is early, so for the first month the phone gets test jobs, not paid work, and earnings are cents. An overnight charge covers a third of the 24 hours we ask for; send one piece of feedback and you are a numbered founding member with a badge you can share. ayni-ai.com/founding

### F5. LinkedIn (founder's account)

We are looking for Ayni's first 1,000 founding members.

Ayni is an open-source network that runs private AI on idle phones and laptops: every job is sealed to one device, so nobody, including us, can read it. Before we ask a single customer to trust it with their data, we want a thousand people to test it for a month.

The deal is plain. Your device gets test jobs, not paid work, for 30 days. You tell us what broke. The network checks both and gives you a numbered badge (#1 to #1,000) you can add to your profile and anyone can verify.

Ten minutes to join: ayni-ai.com/founding

### F6. Personal message

Hi <name>, I am recruiting Ayni's first 1,000 founding members and I would love you to be one of the first. Ayni runs private AI jobs on idle phones and laptops, encrypted so nobody, including me, can read them. The ask: put a phone on its charger or a laptop you are not using on the network for a month (test jobs only, earnings are cents), and tell me one thing that confused you. You get a numbered founding-member badge and my full attention on what you report. Ten minutes: ayni-ai.com/founding. Thank you.

### F7. Student clubs and hackathon organisers (email)

Subject: A month-long open-source project for <club>: founding members of a private-AI network

Hi <name>, Ayni is an open-source (Apache 2.0) network that runs private AI jobs on idle phones and laptops. We are recruiting 1,000 founding members to test it for a month: each member's device gets test jobs, and each member sends one piece of feedback. The coordinator validates members itself and numbers them, with a badge for LinkedIn and GitHub. For a club it is a ready-made project: real distributed-systems code (Go, Rust, llama.cpp, X25519 sealed boxes), a public threat model to pick apart, and a team that answers every issue. Could I do a 20-minute session for your members, or share it in your channel? ayni-ai.com/founding

### F8. Discord (first message, no link)

Building an open-source network that runs llama.cpp jobs sealed to idle phones and laptops (X25519 per job, nothing logged). Looking for founding testers: a month of test jobs on a device you are not using, one piece of feedback, a numbered badge. Happy to share the link and the threat model if anyone wants them.

### F9. Member share text (pre-filled in the app)

I'm founding member #N of Ayni, the open-source network that runs private AI on idle phones and laptops. One of the first 1,000. [share link]

### F10. Day-3 nudge (to enrolled members with a served job and no feedback)

Hi <name>, your device has served Ayni jobs since <date>, thank you. One thing stands between you and your founding-member number: tell us one thing. What confused you, what broke, what you expected. Two lines is plenty: app.ayni-ai.com/testers. We answer within a day.

### F11. GitHub Discussions announcement

**Title:** Founding members: the first 1,000, validated by the network

Today we open founding membership. Enroll at app.ayni-ai.com/founding, keep a device online 24 hours in total during your test month, send one piece of feedback, and the coordinator gives you a permanent number and a badge. Test month means test jobs, not regular paid workloads, and earnings are cents; we say so up front because that is the deal. Progress toward 1,000 members and 100 devices online at any time is live at ayni-ai.com/founding. Every Friday this category gets "You said, we did": what shipped from your feedback, with your name if you want it.

### Reply kit (founding questions)

- **Why no paid work?** Buyers send private data only to a network that is proven. A month of test
  jobs on a growing fleet is how it gets proven; we publish the bar (1,000 members, 100 online at
  any time) and the live numbers.
- **What does the badge prove?** That the coordinator saw your device serve a job, saw it online for
  24 hours and received your feedback. Nobody can grant it by hand, and it is numbered in order.
- **Is this crypto?** No token, no chain. Earnings are dollars, metered per job, paid through Stripe.
- **What happens after 1,000?** The program closes; badges are permanent. Paid workloads ramp as
  buyers arrive, and founding members' devices are already in the fleet when they do.
- **Battery and data?** The phone works only while charging and on Wi-Fi; measured costs:
  ayni-ai.com/faq/device.
- **Can you, or I, read the jobs?** Jobs are sealed to one device with a fresh key; the coordinator
  relays ciphertext and the build fails if any path could log a prompt. On a community-tier device
  the owner could in principle inspect memory; that is why buyers can require attested devices
  (WHITEPAPER, trust tiers).

## Progress log

One row per Monday: `python3 scripts/founding-report.py --row`. Baseline before launch (from
GTM-COMMUNITY-PLAN, 2026-09-10): 3 accounts, 10 devices ever, 1 online.

| Date | Accounts | Enrolled | Served a job | Validated | Floor (7 d) | Peak (7 d) | Notes |
|---|---|---|---|---|---|---|---|
