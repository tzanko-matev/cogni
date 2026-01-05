from __future__ import annotations

from entropy_agent.policy import CommandPolicy


def test_is_safe_allows_known_prefix() -> None:
    policy = CommandPolicy()
    ok, reason = policy.is_safe("python -c 'print(1)'")
    assert ok is True
    assert reason == "ok"


def test_is_safe_blocks_unknown_prefix() -> None:
    policy = CommandPolicy()
    ok, reason = policy.is_safe("echo hi")
    assert ok is False
    assert "allowlist" in reason


def test_is_safe_blocks_denied_pattern() -> None:
    policy = CommandPolicy()
    ok, reason = policy.is_safe("python -c 'print(1)' && rm -rf /")
    assert ok is False
    assert "Denied" in reason
