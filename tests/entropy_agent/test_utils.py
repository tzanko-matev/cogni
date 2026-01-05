from __future__ import annotations

from entropy_agent.utils import normalize_key, read_text, severity_weight, unified_diff, write_text


def test_normalize_key_collapses_whitespace_and_case() -> None:
    assert normalize_key("  Foo--BAR  baz ") == "foo bar baz"


def test_severity_weight_maps_levels() -> None:
    assert severity_weight("low") == 0.2
    assert severity_weight("medium") == 0.5
    assert severity_weight("high") == 0.8
    assert severity_weight("critical") == 1.0


def test_read_write_text_roundtrip(tmp_path) -> None:
    path = tmp_path / "note.txt"
    write_text(path, "hello")
    assert read_text(path) == "hello"


def test_unified_diff_includes_paths() -> None:
    diff = unified_diff("a\n", "b\n", "file.txt")
    assert "a/file.txt" in diff
    assert "b/file.txt" in diff
