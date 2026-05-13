#!/usr/bin/env python3
"""Validate checked-in OpenProse sample outputs."""

from __future__ import annotations

import json
import sys
from pathlib import Path


EXPECTED_FILES = [
    "01-startup_brief.md",
    "02-research_assignment.md",
    "03-research_packet.md",
    "04-ranked_recommendations.md",
    "05-top_pick_explanations.md",
    "06-markdown_report.md",
]

BAD_REPORT_PHRASES = [
    "fallback",
    "top-3",
    "top 3",
    "top scored",
    "top picks",
]


def load_json(path: Path) -> object:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise ValueError(f"{path}: invalid JSON: {exc}") from exc


def validate_example(path: Path) -> list[str]:
    errors: list[str] = []
    for name in EXPECTED_FILES:
        if not (path / name).exists():
            errors.append(f"{path}: missing {name}")
    if errors:
        return errors

    assignment = load_json(path / "02-research_assignment.md")
    packet = load_json(path / "03-research_packet.md")
    ranking = load_json(path / "04-ranked_recommendations.md")
    explanations = load_json(path / "05-top_pick_explanations.md")
    report = (path / "06-markdown_report.md").read_text(encoding="utf-8")

    if not isinstance(assignment, dict):
        errors.append(f"{path}: assignment must be a JSON object")
    if not isinstance(packet, dict):
        errors.append(f"{path}: research packet must be a JSON object")
        return errors
    if not isinstance(ranking, dict):
        errors.append(f"{path}: ranked_recommendations must be a JSON object")
        return errors
    if not isinstance(explanations, list):
        errors.append(f"{path}: top_pick_explanations must be a JSON array")
        return errors

    if packet.get("retrieval", {}).get("no_llm") is not True:
        errors.append(f"{path}: research packet missing retrieval.no_llm=true")

    notes = packet.get("summary", {}).get("notes", [])
    for note in notes if isinstance(notes, list) else []:
        if "ranking is deterministic" in str(note).lower():
            errors.append(f"{path}: packet note still claims CLI ranking")

    grants = packet.get("grants", [])
    if not isinstance(grants, list) or not grants:
        errors.append(f"{path}: research packet must contain candidate grants")
        grants = []
    grant_ids = {str(item.get("recommendation_id", "")) for item in grants if isinstance(item, dict)}
    for item in grants:
        if not isinstance(item, dict):
            errors.append(f"{path}: grant item must be an object")
            continue
        rec_id = str(item.get("recommendation_id", "")).strip()
        if not rec_id:
            errors.append(f"{path}: grant missing recommendation_id")
        if "score" in item:
            errors.append(f"{path}: grant {rec_id} exposes removed CLI score")
        if not item.get("evidence"):
            errors.append(f"{path}: grant {rec_id} missing evidence")

    recommendations = ranking.get("recommendations", [])
    rejected = ranking.get("rejected_candidates", [])
    no_good = ranking.get("no_good_matches")
    if not isinstance(recommendations, list):
        errors.append(f"{path}: recommendations must be an array")
        recommendations = []
    if not isinstance(rejected, list):
        errors.append(f"{path}: rejected_candidates must be an array")
        rejected = []
    if not isinstance(no_good, bool):
        errors.append(f"{path}: no_good_matches must be boolean")
    if no_good is True and recommendations:
        errors.append(f"{path}: no_good_matches=true but recommendations is non-empty")
    if no_good is False and not recommendations:
        errors.append(f"{path}: no_good_matches=false but recommendations is empty")
    if len(recommendations) > 5:
        errors.append(f"{path}: recommendations exceeds cap of 5")

    recommendation_ids: list[str] = []
    for item in recommendations:
        if not isinstance(item, dict):
            errors.append(f"{path}: recommendation item must be an object")
            continue
        rec_id = str(item.get("recommendation_id", "")).strip()
        recommendation_ids.append(rec_id)
        if rec_id not in grant_ids:
            errors.append(f"{path}: recommendation {rec_id!r} not found in research packet")
        for key in ("program_name", "agency", "confidence", "why_this_fits", "caveats", "next_step"):
            if not str(item.get(key, "")).strip():
                errors.append(f"{path}: recommendation {rec_id} missing {key}")

    for item in rejected:
        if not isinstance(item, dict):
            errors.append(f"{path}: rejected candidate item must be an object")
            continue
        rec_id = str(item.get("recommendation_id", "")).strip()
        if rec_id and rec_id not in grant_ids:
            errors.append(f"{path}: rejected candidate {rec_id!r} not found in research packet")
        if not str(item.get("reason", "")).strip():
            errors.append(f"{path}: rejected candidate {rec_id} missing reason")

    explanation_ids = []
    for item in explanations:
        if not isinstance(item, dict):
            errors.append(f"{path}: explanation item must be an object")
            continue
        rec_id = str(item.get("recommendation_id", "")).strip()
        explanation_ids.append(rec_id)
        if item.get("no_llm") is not True:
            errors.append(f"{path}: explanation {rec_id} missing no_llm=true")
        if rec_id not in recommendation_ids:
            errors.append(f"{path}: explanation {rec_id!r} was not selected by ranker")
    if sorted(explanation_ids) != sorted(recommendation_ids):
        errors.append(f"{path}: explanations must match selected recommendation IDs")

    lower_report = report.lower()
    for phrase in BAD_REPORT_PHRASES:
        if phrase in lower_report:
            errors.append(f"{path}: report contains disallowed phrase {phrase!r}")
    if no_good is True and "no credible current recommendation" not in lower_report:
        errors.append(f"{path}: no-good report must say no credible recommendation was found")
    if len(report.splitlines()) > 200:
        errors.append(f"{path}: report exceeds 200 lines")

    return errors


def main() -> int:
    root = Path("examples/openprose/sample-outputs")
    errors: list[str] = []
    for path in sorted(p for p in root.iterdir() if p.is_dir()):
        errors.extend(validate_example(path))
    if errors:
        for error in errors:
            print(f"example-output: {error}", file=sys.stderr)
        return 1
    print("example-output: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
