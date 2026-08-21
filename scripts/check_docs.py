#!/usr/bin/env python3
"""Lightweight documentation structure and relative-link checks."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote


ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
EXPECTED_DOCS = {
    "README.md",
    "engineering/ExtensionDevelopmentGuide.md",
    "engineering/FrontendArchitectureGuide.md",
    "engineering/PlatformEngineeringGuide.md",
    "engineering/ProjectStructureGuide.md",
    "operations/PlatformOperationsGuide.md",
    "user-guide/DataPermissionUserGuide.md",
    "user-guide/FieldTypeGuide.md",
    "user-guide/LinkageConfig.md",
    "user-guide/LowCodeManual.md",
    "user-guide/OrganizationManagementUserGuide.md",
    "user-guide/PlatformAdministrationGuide.md",
    "user-guide/PlatformUserGuide.md",
}
INLINE_LINK_PATTERN = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
REFERENCE_LINK_PATTERN = re.compile(r"^\s*\[[^\]]+\]:\s*(\S+)", re.MULTILINE)
HTML_LINK_PATTERN = re.compile(r"\b(?:href|src)=[\"']([^\"']+)[\"']", re.IGNORECASE)
FENCED_CODE_PATTERN = re.compile(r"```.*?```|~~~.*?~~~", re.DOTALL)
INLINE_CODE_PATTERN = re.compile(r"`[^`\n]+`")
SCHEMES = ("http://", "https://", "mailto:", "tel:", "data:")
FORBIDDEN_REFERENCES = ("docs/" + "_" + "construction", "docs/development")


def markdown_files() -> list[Path]:
    files = [ROOT / "README.md"]
    files.extend(DOCS.rglob("*.md"))
    return sorted(set(files))


def local_target(source: Path, raw_target: str) -> Path | None:
    target = raw_target.strip()
    if not target or target.startswith("#") or target.startswith(SCHEMES):
        return None
    if target.startswith("<") and target.endswith(">"):
        target = target[1:-1]
    if " " in target:
        target = target.split(" ", 1)[0]
    target = unquote(target.split("#", 1)[0].split("?", 1)[0])
    if not target:
        return None
    return (source.parent / target).resolve()


def without_code(text: str) -> str:
    return INLINE_CODE_PATTERN.sub("", FENCED_CODE_PATTERN.sub("", text))


def link_targets(text: str) -> list[str]:
    return [
        *(match.group(1) for match in INLINE_LINK_PATTERN.finditer(text)),
        *(match.group(1) for match in REFERENCE_LINK_PATTERN.finditer(text)),
        *(match.group(1) for match in HTML_LINK_PATTERN.finditer(text)),
    ]


def main() -> int:
    errors: list[str] = []
    files = markdown_files()

    actual_docs = {
        path.relative_to(DOCS).as_posix()
        for path in DOCS.rglob("*")
        if path.is_file() and path.name != ".DS_Store"
    }
    missing = sorted(EXPECTED_DOCS - actual_docs)
    unexpected = sorted(actual_docs - EXPECTED_DOCS)
    if missing:
        errors.append(f"missing required docs: {', '.join(missing)}")
    if unexpected:
        errors.append(f"unexpected docs files: {', '.join(unexpected)}")

    for path in files:
        if not path.exists():
            errors.append(f"missing documentation entry: {path.relative_to(ROOT)}")
            continue
        if path.stat().st_size == 0:
            errors.append(f"empty documentation file: {path.relative_to(ROOT)}")
        text = without_code(path.read_text(encoding="utf-8"))
        for raw_target in link_targets(text):
            target = local_target(path, raw_target)
            if target is not None and not target.exists():
                errors.append(
                    f"broken link: {path.relative_to(ROOT)} -> {raw_target}"
                )
        for forbidden in FORBIDDEN_REFERENCES:
            if forbidden in text:
                errors.append(f"forbidden path: {path.relative_to(ROOT)} -> {forbidden}")

    reference_files = [ROOT / "Makefile", ROOT / "AGENTS.md"]
    for path in reference_files:
        if not path.exists():
            continue
        text = without_code(path.read_text(encoding="utf-8"))
        for forbidden in FORBIDDEN_REFERENCES:
            if forbidden in text:
                errors.append(f"forbidden path: {path.relative_to(ROOT)} -> {forbidden}")

    if errors:
        print("Documentation check failed:")
        for error in errors:
            print(f"- {error}")
        return 1

    print(f"Documentation check passed: {len(files)} Markdown files checked.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
