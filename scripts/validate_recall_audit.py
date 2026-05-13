#!/usr/bin/env python3
"""Audit whether research surfaces known plausible candidates from a seeded ledger."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path


FIT_ORDER = {"low": 0, "medium": 1, "high": 2}
COMMAND_TIMEOUT_SECONDS = 60

CASES = [
    {
        "name": "polyspectra",
        "assignment": "fixtures/polyspectra-assignment.sample.json",
        "expected": [
            {
                "title": "NSF America's Seed Fund: Advanced Manufacturing",
                "source_id": "nsf-seed-fund-topic",
                "min_fit": "high",
            },
            {
                "title": "NASA SBIR 2026 Hydrazine-Compatible Elastomeric Materials",
                "source_id": "sbir-gov-topics",
                "min_fit": "medium",
            },
        ],
    },
    {
        "name": "cypris",
        "assignment": "fixtures/cypris-assignment.sample.json",
        "expected": [
            {
                "title": "NSF America's Seed Fund: Photonics",
                "source_id": "nsf-seed-fund-topic",
                "min_fit": "high",
            },
            {
                "title": "NSF America's Seed Fund: Chemical Technologies",
                "source_id": "nsf-seed-fund-topic",
                "min_fit": "high",
            },
        ],
    },
    {
        "name": "enact-lab",
        "assignment": "fixtures/enact-lab-assignment.sample.json",
        "expected": [
            {
                "title": "Early Stage Testing of Pharmacologic or Neuromodulatory Interventions for Mental Disorders",
                "source_id": "nih-guide",
                "min_fit": "high",
            },
            {
                "title": "First in Human and Early Stage Clinical Trials",
                "source_id": "nih-guide",
                "min_fit": "high",
            },
        ],
        "forbidden_high": ["Generic Small Business Innovation Research Phase I"],
    },
]


def run_json(args: list[str]) -> dict:
    try:
        proc = subprocess.run(
            args,
            check=False,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=COMMAND_TIMEOUT_SECONDS,
        )
    except subprocess.TimeoutExpired as exc:
        raise SystemExit(
            f"command timed out after {COMMAND_TIMEOUT_SECONDS}s: {' '.join(args)}"
        ) from exc
    if proc.returncode != 0:
        raise SystemExit(f"command failed: {' '.join(args)}\n{proc.stderr}")
    try:
        return json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid JSON from {' '.join(args)}: {exc}\n{proc.stdout}") from exc


def candidate_title(candidate: dict) -> str:
    return str(candidate.get("program_name", "")).strip()


def candidate_sources(candidate: dict) -> set[str]:
    return {
        str(item.get("source_id", "")).strip()
        for item in candidate.get("evidence", [])
        if isinstance(item, dict)
    }


def fit_at_least(candidate: dict, minimum: str) -> bool:
    got = str(candidate.get("eligibility_fit", {}).get("level", "")).strip()
    return FIT_ORDER.get(got, -1) >= FIT_ORDER[minimum]


def assert_expected_candidate(case: dict, packet: dict, expected: dict) -> None:
    title = expected["title"]
    matches = [
        candidate
        for candidate in packet.get("grants", [])
        if title.lower() in candidate_title(candidate).lower()
    ]
    if not matches:
        visible = "\n".join(f"- {candidate_title(item)}" for item in packet.get("grants", []))
        raise SystemExit(
            f"recall-audit[{case['name']}]: expected candidate not surfaced: {title}\n"
            f"visible candidates:\n{visible}"
        )

    candidate = matches[0]
    source_id = expected["source_id"]
    if source_id not in candidate_sources(candidate):
        raise SystemExit(
            f"recall-audit[{case['name']}]: {title!r} missing evidence source {source_id!r}; "
            f"got {sorted(candidate_sources(candidate))}"
        )

    min_fit = expected.get("min_fit")
    if min_fit and not fit_at_least(candidate, min_fit):
        got = candidate.get("eligibility_fit", {}).get("level")
        raise SystemExit(
            f"recall-audit[{case['name']}]: {title!r} fit {got!r}, expected >= {min_fit!r}"
        )

    if not candidate.get("evidence"):
        raise SystemExit(f"recall-audit[{case['name']}]: {title!r} missing evidence")


def assert_forbidden_high(case: dict, packet: dict) -> None:
    forbidden = case.get("forbidden_high", [])
    for phrase in forbidden:
        for candidate in packet.get("grants", []):
            if phrase.lower() not in candidate_title(candidate).lower():
                continue
            got = str(candidate.get("eligibility_fit", {}).get("level", "")).strip()
            if got == "high":
                raise SystemExit(
                    f"recall-audit[{case['name']}]: contraindicated candidate was high fit: "
                    f"{candidate_title(candidate)}"
                )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--fixture", default="fixtures/recall-audit-opportunities.sample.json")
    parser.add_argument("--limit", type=int, default=10)
    args = parser.parse_args()

    binary = str(Path(args.binary).resolve())
    fixture = str(Path(args.fixture).resolve())

    with tempfile.TemporaryDirectory() as tmpdir:
        db_path = str(Path(tmpdir) / "recall-audit.sqlite")
        run_json([binary, "debug", "seed-fixture", "--fixture", fixture, "--db", db_path, "--json"])

        for case in CASES:
            packet = run_json([
                binary,
                "research",
                "--assignment",
                case["assignment"],
                "--db",
                db_path,
                "--refresh",
                "off",
                "--semantic",
                "off",
                "--limit",
                str(args.limit),
                "--json",
            ])
            if packet.get("retrieval", {}).get("no_llm") is not True:
                raise SystemExit(f"recall-audit[{case['name']}]: missing retrieval.no_llm=true")
            for candidate in packet.get("grants", []):
                if "score" in candidate:
                    raise SystemExit(
                        f"recall-audit[{case['name']}]: packet leaked removed score field"
                    )
            for expected in case["expected"]:
                assert_expected_candidate(case, packet, expected)
            assert_forbidden_high(case, packet)

    print("recall-audit: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
