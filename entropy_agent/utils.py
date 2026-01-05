from __future__ import annotations

import difflib
import re
import subprocess
from pathlib import Path
from typing import Tuple

from .models import Severity


# --- Utilities ---
def normalize_key(s: str) -> str:
    s = s.lower()
    s = re.sub(r"[^a-z0-9]+", " ", s)
    s = re.sub(r"\s+", " ", s).strip()
    return s


def severity_weight(sev: Severity) -> float:
    return {"low": 0.2, "medium": 0.5, "high": 0.8, "critical": 1.0}[sev]


def unified_diff(old: str, new: str, path: str) -> str:
    old_lines = old.splitlines(keepends=True)
    new_lines = new.splitlines(keepends=True)
    diff = difflib.unified_diff(
        old_lines, new_lines, fromfile=f"a/{path}", tofile=f"b/{path}"
    )
    return "".join(diff)


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8", errors="replace")


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def run_command(cmd: str, cwd: Path, timeout_sec: int) -> Tuple[int, str]:
    proc = subprocess.run(
        cmd,
        cwd=str(cwd),
        shell=True,
        capture_output=True,
        text=True,
        timeout=timeout_sec,
    )
    out = (proc.stdout or "") + (proc.stderr or "")
    return proc.returncode, out
