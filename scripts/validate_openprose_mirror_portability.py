#!/usr/bin/env python3
"""Validate that the OpenProse example can be mirrored under another prefix."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path("examples/openprose")
CANONICAL_PREFIX = "https://github.com/openprose/grant-finder"
LINK_RE = re.compile(r"\]\(([^)]+)\)")

# These source-repo paths do not exist when this directory is mirrored into
# openprose/prose. If docs need to reference them, they must use a canonical
# GitHub URL instead.
SOURCE_ROOT_PATHS = (
    "docs/",
    "schemas/",
    "AGENTS.md",
    "CONTEXT.md",
    "docs/adr/",
)

# Commands should be written from this example directory, which is stable in
# both source and mirror. Root-specific invocations break after subtree add.
ROOT_SPECIFIC_COMMANDS = (
    "prose run examples/openprose/",
    "cat examples/openprose/",
)


def markdown_files() -> list[Path]:
    return sorted(path for path in ROOT.rglob("*.md") if path.is_file())


def validate_links(path: Path, text: str) -> list[str]:
    errors: list[str] = []
    for match in LINK_RE.finditer(text):
        target = match.group(1).strip()
        if not target or target.startswith(("#", "http://", "https://", "mailto:")):
            continue
        if target.startswith("/"):
            errors.append(f"{path}: root-relative link is not mirror-portable: {target}")
        if target.startswith("../"):
            errors.append(f"{path}: link escapes mirrored subtree: {target}")
    return errors


def validate_source_paths(path: Path, text: str) -> list[str]:
    errors: list[str] = []
    for line_no, line in enumerate(text.splitlines(), start=1):
        if CANONICAL_PREFIX in line:
            continue
        for command in ROOT_SPECIFIC_COMMANDS:
            if command in line:
                errors.append(f"{path}:{line_no}: root-specific example command: {command}")
        for source_path in SOURCE_ROOT_PATHS:
            if f"`{source_path}" in line or f"({source_path}" in line:
                errors.append(
                    f"{path}:{line_no}: source-root path needs canonical GitHub URL: {source_path}"
                )
    return errors


def main() -> int:
    errors: list[str] = []
    if not (ROOT / "MIRROR.md").exists():
        errors.append(f"{ROOT / 'MIRROR.md'}: missing mirror provenance note")
    for path in markdown_files():
        text = path.read_text(encoding="utf-8")
        errors.extend(validate_links(path, text))
        errors.extend(validate_source_paths(path, text))
    if errors:
        for error in errors:
            print(f"mirror-portability: {error}", file=sys.stderr)
        return 1
    print("mirror-portability: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
