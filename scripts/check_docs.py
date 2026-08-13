#!/usr/bin/env python3
"""Lightweight documentation structure and relative-link checks."""

from __future__ import annotations

import re
import subprocess
import sys
from collections import defaultdict
from pathlib import Path
from urllib.parse import unquote


ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
ALLOWED_ROOT_FILES = {"README.md", "DocumentationStandard.md"}
FORBIDDEN_DIRECTORIES = {
    DOCS / "analysis",
    DOCS / "report-designer",
    DOCS / "report-v2",
}
INLINE_LINK_PATTERN = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
REFERENCE_LINK_PATTERN = re.compile(r"^\s*\[[^\]]+\]:\s*(\S+)", re.MULTILINE)
HTML_LINK_PATTERN = re.compile(r"\b(?:href|src)=[\"']([^\"']+)[\"']", re.IGNORECASE)
FENCED_CODE_PATTERN = re.compile(r"```.*?```|~~~.*?~~~", re.DOTALL)
INLINE_CODE_PATTERN = re.compile(r"`[^`\n]+`")
SCHEMES = ("http://", "https://", "mailto:", "tel:", "data:")
LEGACY_REFERENCES = {
    "docs/analysis/organization-source/",
    "docs/report-designer/",
    "docs/report-v2/",
    "docs/engineering/platform-capability-backlog-v1.md",
    "docs/Runbook.md",
    "docs/LowCodeManual.md",
    "docs/FieldTypeGuide.md",
    "docs/LinkageConfig.md",
    "docs/DataPermissionUserGuide.md",
    "docs/DataPermissionDesign.md",
    "docs/DataPermissionOwnershipDesign.md",
    "docs/DataPermissionAcceptanceGuide.md",
    "docs/DataPermissionAcceptanceReport.md",
    "docs/DataPermissionFreezeReview.md",
    "docs/AuthenticationArchitectureDesign.md",
    "docs/IntegrationFoundationDesign.md",
    "docs/IntegrationConfigurationDesign.md",
    "docs/IntegrationRuntimeDesign.md",
    "docs/IntegrationRetryDesign.md",
    "docs/IntegrationSyncDesign.md",
    "docs/OrganizationHRSyncDesign.md",
}


def markdown_files() -> list[Path]:
    files = [ROOT / "README.md"]
    files.extend(
        path
        for path in DOCS.rglob("*.md")
        if "development" not in path.relative_to(DOCS).parts
    )
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

    root_files = {path.name for path in DOCS.iterdir() if path.is_file() and path.name != ".DS_Store"}
    unexpected = sorted(root_files - ALLOWED_ROOT_FILES)
    if unexpected:
        errors.append(f"unexpected docs root files: {', '.join(unexpected)}")

    for directory in sorted(FORBIDDEN_DIRECTORIES):
        if directory.exists():
            errors.append(f"legacy docs directory still exists: {directory.relative_to(ROOT)}")

    names: dict[str, list[Path]] = defaultdict(list)
    for path in files:
        if not path.exists():
            errors.append(f"missing documentation entry: {path.relative_to(ROOT)}")
            continue
        if path.stat().st_size == 0:
            errors.append(f"empty documentation file: {path.relative_to(ROOT)}")
        if path.name.lower() != "readme.md":
            names[path.name.casefold()].append(path)
        text = without_code(path.read_text(encoding="utf-8"))
        for raw_target in link_targets(text):
            target = local_target(path, raw_target)
            if target is not None and not target.exists():
                errors.append(
                    f"broken link: {path.relative_to(ROOT)} -> {raw_target}"
                )
        for legacy in sorted(LEGACY_REFERENCES):
            if legacy in text:
                errors.append(f"legacy path: {path.relative_to(ROOT)} -> {legacy}")

        if DOCS in path.parents and "_construction/analysis" not in path.as_posix():
            ignored = subprocess.run(
                ["git", "check-ignore", "-q", str(path.relative_to(ROOT))],
                cwd=ROOT,
                check=False,
            )
            if ignored.returncode == 0:
                errors.append(f"governed documentation is ignored: {path.relative_to(ROOT)}")

    reference_files = [ROOT / "Makefile", ROOT / "AGENTS.md"]
    for path in reference_files:
        if not path.exists():
            continue
        text = without_code(path.read_text(encoding="utf-8"))
        for legacy in sorted(LEGACY_REFERENCES):
            if legacy in text:
                errors.append(f"legacy path: {path.relative_to(ROOT)} -> {legacy}")

    for name, paths in sorted(names.items()):
        if len(paths) > 1:
            rendered = ", ".join(str(path.relative_to(ROOT)) for path in paths)
            errors.append(f"duplicate documentation name {name}: {rendered}")

    if errors:
        print("Documentation check failed:")
        for error in errors:
            print(f"- {error}")
        return 1

    print(f"Documentation check passed: {len(files)} Markdown files checked.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
