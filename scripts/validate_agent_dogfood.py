#!/usr/bin/env python3
"""Dogfood the Grant Finder CLI as an upstream agent would."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path


def load_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise SystemExit(f"{path}: could not parse JSON: {exc}") from exc


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


def validate_schema_subset(schema: dict, value: object, path: str) -> None:
    """Validate the schema features this repo uses, without external deps."""
    if "type" in schema and not schema_type_matches(schema["type"], value):
        raise SystemExit(f"{path}: expected type {schema['type']!r}, got {type(value).__name__}")

    if "enum" in schema and value not in schema["enum"]:
        raise SystemExit(f"{path}: expected one of {schema['enum']!r}, got {value!r}")

    if isinstance(value, str) and "minLength" in schema and len(value) < int(schema["minLength"]):
        raise SystemExit(f"{path}: expected minLength {schema['minLength']}, got {len(value)}")

    if isinstance(value, int) and not isinstance(value, bool) and "minimum" in schema:
        if value < int(schema["minimum"]):
            raise SystemExit(f"{path}: expected minimum {schema['minimum']}, got {value}")

    if isinstance(value, dict):
        for key in schema.get("required", []):
            if key not in value:
                raise SystemExit(f"{path}: missing required field {key!r}")
        properties = schema.get("properties", {})
        for key, subschema in properties.items():
            if key in value:
                validate_schema_subset(subschema, value[key], f"{path}.{key}")

    if isinstance(value, list) and "items" in schema:
        for i, item in enumerate(value):
            validate_schema_subset(schema["items"], item, f"{path}[{i}]")


def schema_type_matches(expected: object, value: object) -> bool:
    if isinstance(expected, list):
        return any(schema_type_matches(item, value) for item in expected)
    if expected == "object":
        return isinstance(value, dict)
    if expected == "array":
        return isinstance(value, list)
    if expected == "string":
        return isinstance(value, str)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "boolean":
        return isinstance(value, bool)
    if expected == "null":
        return value is None
    return True


def assert_research_contract(packet: dict, limit: int) -> None:
    if not packet.get("retrieval", {}).get("no_llm"):
        raise SystemExit("research packet did not record no_llm=true")

    grants = packet.get("grants", [])
    if not grants:
        raise SystemExit("research returned no grants")
    if len(grants) > limit:
        raise SystemExit(f"research returned {len(grants)} grants, expected <= {limit}")

    seen_recommendations: set[str] = set()
    for grant in grants:
        rec_id = str(grant.get("recommendation_id", "")).strip()
        if not rec_id:
            raise SystemExit("recommendation missing recommendation_id")
        if rec_id in seen_recommendations:
            raise SystemExit(f"duplicate recommendation_id: {rec_id}")
        seen_recommendations.add(rec_id)

        if not str(grant.get("url", "")).strip():
            raise SystemExit(f"{rec_id}: recommendation missing URL")
        evidence = grant.get("evidence", [])
        if not evidence:
            raise SystemExit(f"{rec_id}: recommendation has no evidence")
        for item in evidence:
            for key in ("source_id", "url", "claim"):
                if not str(item.get(key, "")).strip():
                    raise SystemExit(f"{rec_id}: evidence item missing {key}")

    first = grants[0]
    if first.get("eligibility_fit", {}).get("level") not in {"high", "medium"}:
        raise SystemExit(f"unexpected fit level: {first.get('eligibility_fit')}")

    coverage = packet.get("coverage", [])
    arpa = [row for row in coverage if row.get("source_lane") == "ARPA-E"]
    if not arpa or "No current ARPA-E programs match" not in arpa[0].get("note", ""):
        raise SystemExit("missing ARPA-E negative evidence row")


def assert_explain_contract(explanation: dict) -> None:
    if not explanation.get("no_llm"):
        raise SystemExit("explain did not record no_llm=true")
    if not explanation.get("evidence"):
        raise SystemExit("explain returned no evidence")
    if not explanation.get("sources"):
        raise SystemExit("explain returned no source trail")


def assert_status_contract(status: dict) -> None:
    if not status.get("no_llm"):
        raise SystemExit("status did not record no_llm=true")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--assignment", default="fixtures/acme-deeptech-assignment.sample.json")
    parser.add_argument("--fixture", default="fixtures/acme-deeptech-opportunities.sample.json")
    args = parser.parse_args()

    binary = str(Path(args.binary).resolve())
    assignment_path = Path(args.assignment).resolve()
    assignment = str(assignment_path)
    fixture = str(Path(args.fixture).resolve())
    repo = Path.cwd()

    assignment_schema = load_json(repo / "schemas/research-assignment.schema.json")
    packet_schema = load_json(repo / "schemas/research-packet.schema.json")
    validate_schema_subset(assignment_schema, load_json(assignment_path), "assignment")

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
        validate_schema_subset(packet_schema, packet, "research_packet")
        assert_research_contract(packet, limit=10)
        first = packet["grants"][0]

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
        assert_explain_contract(explanation)

        status = run_json([binary, "status", "--assignment", assignment, "--db", db_path, "--json"])
        assert_status_contract(status)

    print("agent-dogfood: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
