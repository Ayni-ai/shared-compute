#!/usr/bin/env python3
"""
Founding-member progress report (docs/FOUNDING-MEMBERS.md): where the program stands
against its two goals, 1,000 validated members and 100 devices online at any time,
and which stop/go rule (if any) says to fix something before posting more.

  python3 scripts/founding-report.py              # print the report
  python3 scripts/founding-report.py --record     # also append a snapshot to the history
  python3 scripts/founding-report.py --row        # one Markdown row for the progress log
  python3 scripts/founding-report.py --from f.json  # report on a saved /admin/founding body

Reads SC_ADMIN_TOKEN from the repo's .env.local (or the environment). Standard library
only. History lives in ~/.ayni/founding/history.jsonl. Counts only, never content.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
API = os.environ.get("AYNI_BASE_URL", "https://api.ayni-ai.com")
HISTORY = os.path.expanduser("~/.ayni/founding/history.jsonl")

# Stop/go thresholds, mirrored from docs/FOUNDING-MEMBERS.md ("Stop/go rules").
MIN_VALIDATION_RATE = 0.25   # validated / enrolled, once 50 have enrolled for a week
MIN_FLOOR_PER_MEMBER = 0.07  # fleet floor / validated members, once there are 100 members
MIN_FEEDBACK_RATE = 0.60     # gave feedback / served a job


def load_env() -> None:
    p = os.path.join(REPO, ".env.local")
    if not os.path.exists(p):
        return
    with open(p, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))


def fetch() -> dict:
    token = os.environ.get("SC_ADMIN_TOKEN", "")
    if not token:
        sys.exit("SC_ADMIN_TOKEN is not set (put it in .env.local)")
    req = urllib.request.Request(API + "/admin/founding", headers={"Authorization": "Bearer " + token})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return json.loads(r.read())
    except urllib.error.HTTPError as e:
        if e.code == 503:
            sys.exit("the coordinator cannot reach its database right now (503); nothing recorded, try again later")
        raise


def pct(a: int, b: int) -> str:
    return f"{(100 * a / b):.0f}%" if b else "—"


def last_week(history: list[dict]) -> dict | None:
    """The newest snapshot at least six days old, for week-over-week change."""
    now = datetime.now(timezone.utc)
    for h in reversed(history):
        at = datetime.fromisoformat(h["at"])
        if (now - at).days >= 6:
            return h
    return None


def read_history() -> list[dict]:
    try:
        with open(HISTORY, encoding="utf-8") as f:
            return [json.loads(l) for l in f if l.strip()]
    except FileNotFoundError:
        return []


def snapshot(d: dict) -> dict:
    f = d.get("funnel", {})
    fleet = d.get("fleet_7d", {})
    return {
        "at": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "users": f.get("users", 0),
        "enrolled": f.get("enrolled", 0),
        "validated": f.get("validated", 0),
        "enrolled_7d": f.get("enrolled_7d", 0),
        "validated_7d": f.get("validated_7d", 0),
        "online_now": d.get("online_now", 0),
        "floor_7d": fleet.get("floor", 0),
        "peak_7d": fleet.get("peak", 0),
    }


def flags(d: dict) -> list[str]:
    f = d.get("funnel", {})
    fleet = d.get("fleet_7d", {})
    out = []
    enrolled, validated = f.get("enrolled", 0), f.get("validated", 0)
    older = enrolled - f.get("enrolled_7d", 0)  # enrolled more than a week ago: had time to finish
    if older >= 50 and validated / max(1, older) < MIN_VALIDATION_RATE:
        out.append(f"STOP posting: only {pct(validated, older)} of members enrolled 7+ days ago are validated "
                   f"(rule: {MIN_VALIDATION_RATE:.0%}). Find the step where they stall below and fix it first.")
    served, fb = f.get("served_job", 0), f.get("gave_feedback", 0)
    if served >= 20 and fb / served < MIN_FEEDBACK_RATE:
        out.append(f"Feedback is the bottleneck ({pct(fb, served)} of members with a served job wrote to us): "
                   "send the day-3 nudge and put the form in front of them.")
    if validated >= 100 and fleet.get("floor", 0) / validated < MIN_FLOOR_PER_MEMBER:
        out.append(f"Retention problem: floor {fleet.get('floor', 0)} for {validated} members "
                   f"(< {MIN_FLOOR_PER_MEMBER} per member). Members leave after validating; fix the reason to stay "
                   "(pulse jobs, founding pool, real workloads) before recruiting harder.")
    if fleet.get("covered_hours", 0) < 24 * 6:
        out.append(f"The 7-day floor covers {fleet.get('covered_hours', 0)} hours of samples so far; read it as provisional.")
    return out


def report(d: dict, history: list[dict]) -> str:
    f = d.get("funnel")
    targets = d.get("targets", {"members": 1000, "online_floor": 100})
    rules = d.get("rules", {})
    if f is None:
        return "Accounts are not enabled on this coordinator: no founding funnel to report."
    fleet7, fleet24 = d.get("fleet_7d", {}), d.get("fleet_24h", {})
    lines = [
        f"Ayni founding members — {datetime.now(timezone.utc):%Y-%m-%d %H:%M} UTC",
        "",
        f"  Validated members   {f['validated']:>5} / {targets['members']}   (+{f['validated_7d']} this week)",
        f"  Online at any time  {fleet7.get('floor', 0):>5} / {targets['online_floor']}   "
        f"(7-day floor; median {fleet7.get('median', 0)}, peak {fleet7.get('peak', 0)}, now {d.get('online_now', 0)})",
        f"  Last 24 h           floor {fleet24.get('floor', 0)}, peak {fleet24.get('peak', 0)}",
        "",
        "  Funnel (enrolled members)",
        f"    signed up (all accounts)   {f['users']}",
        f"    enrolled in test month     {f['enrolled']}   (+{f['enrolled_7d']} this week, {f['referred_enrolled']} via invites)",
        f"    linked a device            {f['with_device']}   {pct(f['with_device'], f['enrolled'])}",
        f"    a device served a job      {f['served_job']}   {pct(f['served_job'], f['enrolled'])}",
        f"    {rules.get('min_online_hours', 24)} h online in total       {f['online_enough']}   {pct(f['online_enough'], f['enrolled'])}",
        f"    sent feedback              {f['gave_feedback']}   {pct(f['gave_feedback'], f['enrolled'])}",
        f"    validated                  {f['validated']}   {pct(f['validated'], f['enrolled'])}",
    ]
    pace = f["validated_7d"]
    left = targets["members"] - f["validated"]
    if left <= 0:
        lines += ["", "  The founding program is full."]
    elif pace > 0:
        lines += ["", f"  At this week's pace ({pace}/week) the program fills in about {-(-left // pace)} weeks."]
    else:
        lines += ["", "  No members validated this week."]
    prev = last_week(history)
    if prev:
        lines.append(f"  Week over week: enrolled {prev['enrolled']} → {f['enrolled']}, validated {prev['validated']} → "
                     f"{f['validated']}, floor {prev['floor_7d']} → {fleet7.get('floor', 0)}.")
    fl = flags(d)
    if fl:
        lines += ["", "  Rules triggered:"] + [f"    - {x}" for x in fl]
    return "\n".join(lines)


def row(d: dict) -> str:
    f, fleet = d.get("funnel", {}), d.get("fleet_7d", {})
    return (f"| {datetime.now(timezone.utc):%Y-%m-%d} | {f.get('users', 0)} | {f.get('enrolled', 0)} | {f.get('served_job', 0)} | "
            f"{f.get('validated', 0)} | {fleet.get('floor', 0)} | {fleet.get('peak', 0)} | |")


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--record", action="store_true", help="append a snapshot to the history")
    ap.add_argument("--row", action="store_true", help="print one Markdown row for the progress log")
    ap.add_argument("--from", dest="src", help="read a saved /admin/founding JSON body instead of the API")
    args = ap.parse_args()
    load_env()
    if args.src:
        with open(args.src, encoding="utf-8") as fh:
            d = json.load(fh)
    else:
        d = fetch()
    history = read_history()
    if args.row:
        print(row(d))
        return
    if d.get("degraded"):
        failed = ", ".join(d.get("errors") or []) or "unknown"
        print(f"WARNING: the coordinator could not read part of its database ({failed}).")
        if d.get("funnel") is None or args.record:
            sys.exit("Report skipped and nothing recorded: try again once /health answers 200.")
        print("The numbers below are incomplete.\n")
    print(report(d, history))
    if args.record and d.get("funnel") is not None:
        os.makedirs(os.path.dirname(HISTORY), exist_ok=True)
        with open(HISTORY, "a", encoding="utf-8") as fh:
            fh.write(json.dumps(snapshot(d)) + "\n")


if __name__ == "__main__":
    main()
