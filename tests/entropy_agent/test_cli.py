from __future__ import annotations

from entropy_agent import tui
from entropy_agent.cli import prompt_for_goal


def test_prompt_for_goal_retries_on_empty(monkeypatch) -> None:
    inputs = iter(["", "  ", "Ship the feature"])
    monkeypatch.setattr(tui, "_HAS_PT", False)
    monkeypatch.setattr(tui, "_raw_input", lambda _: next(inputs))
    assert prompt_for_goal() == "Ship the feature"
