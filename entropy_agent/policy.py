from __future__ import annotations

import re
import shlex
from typing import Tuple


# --- Safe command policy ---
class CommandPolicy:
    """
    Conservative defaults. You can loosen them as needed.
    """

    # Allowlist by command prefix (first token)
    allowed_prefixes = {
        "python", "python3", "pytest", "pip", "pip3",
        "ruff", "black", "mypy",
        "node", "npm", "pnpm",
        "git",
    }

    # Disallow obvious footguns
    denied_patterns = [
        r"\brm\b",
        r"\bsudo\b",
        r"\bshutdown\b",
        r"\breboot\b",
        r"\bmkfs\b",
        r"\b:\(\)\s*\{",  # fork bomb-ish
        r"\bchmod\s+777\b",
        r"\bcurl\b",
        r"\bwget\b",
    ]

    def is_safe(self, cmd: str) -> Tuple[bool, str]:
        cmd = cmd.strip()
        if not cmd:
            return False, "Empty command"
        first = shlex.split(cmd)[0]
        if first not in self.allowed_prefixes:
            return False, f"Command '{first}' not in allowlist"
        for pat in self.denied_patterns:
            if re.search(pat, cmd):
                return False, f"Denied by pattern: {pat}"
        return True, "ok"
