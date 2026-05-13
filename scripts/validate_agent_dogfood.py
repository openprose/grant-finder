#!/usr/bin/env python3
"""Dogfood the Grant Finder CLI as an upstream agent would."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path


def run_json(args: list[str]) -> dict:
    proc = subprocess.run(args, check=False, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if proc.returncode != 0:
        raise SystemExit(f"command failed: {' '.join(args)}\n{proc.stderr}")
    try:
        return json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid JSON from {' '.join(args)}: {exc}\n{proc.stdout}") from exc


def run_text(args: list[str]) -> str:
    proc = subprocess.run(args, check=False, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if proc.returncode != 0:
        raise SystemExit(f"command failed: {' '.join(args)}\n{proc.stderr}")
    return proc.stdout


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--assignment", default="fixtures/acme-deeptech-assignment.sample.json")
    parser.add_argument("--fixture", default="fixtures/acme-deeptech-opportunities.sample.json")
    args = parser.parse_args()

    binary = str(Path(args.binary).resolve())
    assignment = str(Path(args.assignment).resolve())
    fixture = str(Path(args.fixture).resolve())

    help_text = run_text([binary, "--help"])
    for command in ("research", "explain", "status"):
        if command not in help_text:
            raise SystemExit(f"missing public command in --help: {command}")
    for command in (" sync", " feeds", " grants", " federal-register", " sql"):
        if command in help_text:
            raise SystemExit(f"source plumbing leaked into top-level help: {command.strip()}")

    with tempfile.TemporaryDirectory() as tmpdir:
        db_path = str(Path(tmpdir) / "grant-finder.sqlite")
        run_json([binary, "debug", "seed-fixture", "--fixture", fixture, "--db", db_path, "--json"])

        packet = run_json([
            binary,
            "research",
            "--assignment",
            assignment,
            "--db",
            db_path,
            "--refresh",
            "off",
            "--semantic",
            "usearch",
            "--json",
        ])
        if not packet.get("retrieval", {}).get("no_llm"):
            raise SystemExit("research packet did not record no_llm=true")
        grants = packet.get("grants", [])
        if not grants:
            raise SystemExit("research returned no grants")
        first = grants[0]
        if first.get("eligibility_fit", {}).get("level") not in {"high", "medium"}:
            raise SystemExit(f"unexpected fit level: {first.get('eligibility_fit')}")
        if not first.get("evidence"):
            raise SystemExit("recommendation has no evidence")
        coverage = packet.get("coverage", [])
        arpa = [row for row in coverage if row.get("source_lane") == "ARPA-E"]
        if not arpa or "No current ARPA-E programs match" not in arpa[0].get("note", ""):
            raise SystemExit("missing ARPA-E negative evidence row")

        selected = run_json([
            binary,
            "research",
            "--assignment",
            assignment,
            "--db",
            db_path,
            "--refresh",
            "off",
            "--semantic",
            "usearch",
            "--select",
            "retrieval.backend,grants.program_name",
            "--compact",
        ])
        if sorted(selected) != ["grants.program_name", "retrieval.backend"]:
            raise SystemExit(f"--select returned unexpected keys: {sorted(selected)}")
        if not selected["grants.program_name"]:
            raise SystemExit("--select grants.program_name returned no values")

        explanation = run_json([binary, "explain", first["recommendation_id"], "--db", db_path, "--json"])
        if not explanation.get("no_llm"):
            raise SystemExit("explain did not record no_llm=true")
        if not explanation.get("evidence"):
            raise SystemExit("explain returned no evidence")

        status = run_json([binary, "status", "--assignment", assignment, "--db", db_path, "--json"])
        if not status.get("no_llm"):
            raise SystemExit("status did not record no_llm=true")

    print("agent-dogfood: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
