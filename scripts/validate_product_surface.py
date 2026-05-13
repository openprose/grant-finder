#!/usr/bin/env python3
"""Validate the Grant Finder product surface contract."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path


def load_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:  # pragma: no cover - diagnostic path
        raise SystemExit(f"{path}: could not parse JSON: {exc}") from exc


def command_names(surface: dict, key: str) -> list[str]:
    return [str(item.get("name", "")).strip() for item in surface.get(key, [])]


def validate_contract(surface: dict, repo: Path) -> list[str]:
    errors: list[str] = []

    required_keys = {
        "schema_version",
        "product_name",
        "primary_persona",
        "primary_command",
        "public_commands",
        "background_capabilities",
        "prohibited_public_commands",
        "acceptance_artifacts",
    }
    missing = sorted(required_keys - set(surface))
    if missing:
        errors.append(f"surface missing keys: {', '.join(missing)}")

    public = command_names(surface, "public_commands")
    if surface.get("primary_command") not in public:
        errors.append("primary_command must appear in public_commands")

    expected_public = {"research", "explain", "status"}
    missing_public = sorted(expected_public - set(public))
    if missing_public:
        errors.append(f"public command set missing: {', '.join(missing_public)}")

    prohibited = set(surface.get("prohibited_public_commands", []))
    exposed_prohibited = sorted(prohibited.intersection(public))
    if exposed_prohibited:
        errors.append(f"prohibited commands listed as public: {', '.join(exposed_prohibited)}")

    for item in surface.get("public_commands", []):
        if item.get("role") != "agent-facing":
            errors.append(f"public command {item.get('name')!r} must have role agent-facing")
        if not item.get("input") or not item.get("output"):
            errors.append(f"public command {item.get('name')!r} must define input and output")

    allowed_automation = {"automatic", "internal", "debug-only"}
    for item in surface.get("background_capabilities", []):
        automation = item.get("automation")
        if automation not in allowed_automation:
            errors.append(
                f"background capability {item.get('name')!r} has invalid automation {automation!r}"
            )

    for rel in surface.get("acceptance_artifacts", []):
        if not (repo / rel).exists():
            errors.append(f"acceptance artifact missing: {rel}")

    for rel in surface.get("sample_assignments", []):
        path = repo / rel
        if not path.exists():
            errors.append(f"sample assignment missing: {rel}")
            continue
        try:
            sample = load_json(path)
        except SystemExit as exc:
            errors.append(str(exc))
            continue
        profile = sample.get("company_profile", {})
        if not sample.get("assignment_id"):
            errors.append(f"{rel}: missing assignment_id")
        if not isinstance(profile, dict) or not profile.get("description"):
            errors.append(f"{rel}: missing company_profile.description")
        for key in ("focus_areas", "target_geographies", "known_grants"):
            if key not in sample:
                errors.append(f"{rel}: missing {key}")

    return errors


def parse_help_commands(help_text: str) -> set[str]:
    commands: set[str] = set()
    in_commands = False
    for line in help_text.splitlines():
        stripped = line.strip()
        if stripped == "Available Commands:":
            in_commands = True
            continue
        if in_commands and not stripped:
            continue
        if in_commands and not line.startswith("  "):
            break
        if in_commands:
            commands.add(stripped.split()[0])
    return commands


def validate_cli(binary: Path, surface: dict) -> list[str]:
    errors: list[str] = []
    proc = subprocess.run(
        [str(binary), "--help"],
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=30,
    )
    if proc.returncode != 0:
        return [f"{binary}: --help failed: {proc.stderr.strip()}"]

    actual = parse_help_commands(proc.stdout)
    expected = set(command_names(surface, "public_commands"))
    missing = sorted(expected - actual)
    if missing:
        errors.append(f"CLI missing public commands: {', '.join(missing)}")

    prohibited = set(surface.get("prohibited_public_commands", []))
    exposed = sorted(prohibited.intersection(actual))
    if exposed:
        errors.append(f"CLI exposes prohibited product commands: {', '.join(exposed)}")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--surface", default="docs/product-surface.json")
    parser.add_argument("--repo", default=".")
    parser.add_argument("--check-cli", help="Optional compiled CLI binary to check against the surface")
    args = parser.parse_args()

    repo = Path(args.repo).resolve()
    surface_path = (repo / args.surface).resolve()
    surface = load_json(surface_path)

    errors = validate_contract(surface, repo)
    if args.check_cli:
        errors.extend(validate_cli(Path(args.check_cli).resolve(), surface))

    if errors:
        for error in errors:
            print(f"product-surface: {error}", file=sys.stderr)
        return 1

    print("product-surface: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
