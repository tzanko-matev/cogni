from __future__ import annotations

from typing import Iterable, List, Optional

try:
    from prompt_toolkit.shortcuts import input_dialog, message_dialog, radiolist_dialog, yes_no_dialog

    _HAS_PT = True
except Exception:
    _HAS_PT = False

_raw_input = input


def prompt_required_text(prompt: str, *, multiline: bool = False) -> str:
    while True:
        value = prompt_text(prompt, multiline=multiline).strip()
        if value:
            return value


def prompt_text(prompt: str, *, multiline: bool = False) -> str:
    if _HAS_PT:
        result = input_dialog(title="Input", text=prompt, multiline=multiline).run()
        return (result or "").strip()
    return _raw_input(prompt).strip()


def prompt_yes_no(prompt: str, *, default: bool = False) -> bool:
    if _HAS_PT:
        result = yes_no_dialog(title="Confirm", text=prompt).run()
        if result is None:
            return default
        return bool(result)
    ans = _raw_input(f"{prompt} (y/n): ").strip().lower()
    if not ans:
        return default
    return ans.startswith("y")


def prompt_choice(prompt: str, choices: List[str]) -> str:
    if not choices:
        return ""
    if _HAS_PT:
        result = radiolist_dialog(
            title="Choose",
            text=prompt,
            values=[(c, c) for c in choices],
        ).run()
        if result is None:
            return choices[0]
        return result
    while True:
        print(prompt)
        for i, ch in enumerate(choices, 1):
            print(f"  {i}. {ch}")
        ans = _raw_input("Choose number: ").strip()
        try:
            idx = int(ans)
            if 1 <= idx <= len(choices):
                return choices[idx - 1]
        except Exception:
            pass


def prompt_continue(prompt: str = "Press Enter to continue...") -> None:
    if _HAS_PT:
        message_dialog(title="Continue", text=prompt).run()
        return
    _raw_input(prompt)
