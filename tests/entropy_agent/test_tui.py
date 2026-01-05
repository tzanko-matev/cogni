from __future__ import annotations

from entropy_agent import tui


def test_prompt_required_text_retries_on_empty(monkeypatch) -> None:
    inputs = iter(["", "  ", "ok"])
    monkeypatch.setattr(tui, "_HAS_PT", False)
    monkeypatch.setattr(tui, "_raw_input", lambda _: next(inputs))
    assert tui.prompt_required_text("Enter:") == "ok"


def test_prompt_yes_no_fallback(monkeypatch) -> None:
    inputs = iter(["y"])
    monkeypatch.setattr(tui, "_HAS_PT", False)
    monkeypatch.setattr(tui, "_raw_input", lambda _: next(inputs))
    assert tui.prompt_yes_no("Proceed?") is True


def test_prompt_choice_fallback(monkeypatch) -> None:
    inputs = iter(["2"])
    monkeypatch.setattr(tui, "_HAS_PT", False)
    monkeypatch.setattr(tui, "_raw_input", lambda _: next(inputs))
    assert tui.prompt_choice("Pick", ["a", "b", "c"]) == "b"
