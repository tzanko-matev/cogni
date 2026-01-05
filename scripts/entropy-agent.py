#!/usr/bin/env python3
"""
Entropy-aware agent for large coding tasks.

Core idea:
- Keep the model in low-entropy regions by forcing work into constrained artifacts:
  * definition-of-done
  * risk register / unknowns
  * experiments (tests, prototypes, commands)
  * small implementation steps + verification
- Detect "high entropy" areas via:
  * model-reported confidence
  * disagreement across multiple independently sampled risk registers
- Resolve high-entropy areas via:
  * human-in-the-loop questions, OR
  * runnable experiments that produce evidence

Requires:
  pip install openai pydantic rich
Environment:
  export OPENAI_API_KEY="..."
"""

from __future__ import annotations

import argparse
import dataclasses
import difflib
import json
import os
import re
import shlex
import subprocess
import sys
import textwrap
import time
from pathlib import Path
from typing import Any, Dict, List, Literal, Optional, Tuple

from pydantic import BaseModel, Field, ValidationError

# --- Optional nicer console output ---
try:
    from rich.console import Console
    from rich.panel import Panel
    from rich.text import Text

    console = Console()

    def print_panel(title: str, body: str) -> None:
        console.print(Panel(body, title=title, expand=False))

    def print_info(msg: str) -> None:
        console.print(f"[bold cyan]INFO[/bold cyan] {msg}")

    def print_warn(msg: str) -> None:
        console.print(f"[bold yellow]WARN[/bold yellow] {msg}")

    def print_err(msg: str) -> None:
        console.print(f"[bold red]ERROR[/bold red] {msg}")

except Exception:
    console = None

    def print_panel(title: str, body: str) -> None:
        print(f"\n== {title} ==\n{body}\n")

    def print_info(msg: str) -> None:
        print(f"[INFO] {msg}")

    def print_warn(msg: str) -> None:
        print(f"[WARN] {msg}")

    def print_err(msg: str) -> None:
        print(f"[ERROR] {msg}")


# --- OpenAI SDK wrapper (Responses API) ---
def _require_openai() -> Any:
    try:
        from openai import OpenAI  # type: ignore
        return OpenAI
    except Exception as e:
        raise RuntimeError(
            "openai package not found. Install with: pip install openai"
        ) from e


class OpenAIBackend:
    """
    Uses OpenAI Responses API via the official Python SDK.

    We use responses.parse(...) (Structured Outputs) for schema-constrained steps.
    Fall back to responses.create(...) if parse is unavailable (older SDK).
    """

    def __init__(self) -> None:
        OpenAI = _require_openai()
        self.client = OpenAI()

    def parse(
        self,
        *,
        model: str,
        reasoning_effort: Literal["none", "low", "medium", "high", "xhigh"] = "medium",
        temperature: Optional[float] = None,
        max_output_tokens: int = 2000,
        system: str,
        user: str,
        schema: type[BaseModel],
    ) -> BaseModel:
        def _schema_hint() -> str:
            if schema.__name__ == "StepPlan":
                return json.dumps(
                    {
                        "step_goal": "Concise goal",
                        "rationale": "Why this step is needed",
                        "file_writes": [
                            {"path": "path/to/file.ext", "content": "", "mode": "overwrite"}
                        ],
                        "commands": [{"cmd": "python -m pytest", "purpose": "Run tests", "timeout_sec": 300}],
                        "expected_outcomes": ["What should be true after the step"],
                        "new_tasks": [],
                        "notes": [],
                    },
                    indent=2,
                )
            try:
                return json.dumps(schema.model_json_schema(), indent=2)[:3000]
            except Exception:
                return f"Schema name: {schema.__name__}"

        def _input_items(content: str) -> List[Dict[str, str]]:
            return [
                {"role": "system", "content": system},
                {"role": "user", "content": content},
            ]

        def _extract_json_object(raw: str) -> str:
            text = raw.strip()
            if text.startswith("{") and text.endswith("}"):
                return text
            start = text.find("{")
            end = text.rfind("}")
            if start != -1 and end != -1 and end > start:
                return text[start : end + 1]
            return text

        def _ensure_list(value: Any) -> List[Any]:
            if value is None:
                return []
            if isinstance(value, list):
                return value
            return [value]

        def _coerce_for_schema(data: Any) -> Any:
            if schema.__name__ != "StepPlan" or not isinstance(data, dict):
                return data

            out = dict(data)
            if "step_goal" not in out or not isinstance(out.get("step_goal"), str):
                for key in ("objective", "goal", "step_id", "title"):
                    if isinstance(out.get(key), str):
                        out["step_goal"] = out[key]
                        break
            if "rationale" not in out or not isinstance(out.get("rationale"), str):
                for key in ("rationale", "why", "objective", "notes"):
                    if isinstance(out.get(key), str):
                        out["rationale"] = out[key]
                        break
                out.setdefault("rationale", "")

            files = out.get("file_writes")
            if files is None and "files" in out:
                files = out.get("files")
            files_list = _ensure_list(files)
            coerced_files = []
            for fw in files_list:
                if not isinstance(fw, dict):
                    continue
                path = fw.get("path") or fw.get("file") or fw.get("name")
                content = fw.get("content", "")
                mode = fw.get("mode", "overwrite")
                if path:
                    coerced_files.append({"path": path, "content": content or "", "mode": mode})
            if coerced_files:
                out["file_writes"] = coerced_files

            cmds = out.get("commands") or out.get("cmds")
            cmds_list = _ensure_list(cmds)
            coerced_cmds = []
            for cmd in cmds_list:
                if not isinstance(cmd, dict):
                    continue
                cmd_str = cmd.get("cmd") or cmd.get("command")
                if not cmd_str:
                    continue
                purpose = cmd.get("purpose") or cmd.get("why") or "Run command"
                timeout_sec = cmd.get("timeout_sec", 300)
                coerced_cmds.append(
                    {"cmd": cmd_str, "purpose": purpose, "timeout_sec": timeout_sec}
                )
            if coerced_cmds:
                out["commands"] = coerced_cmds

            if "expected_outcomes" not in out and "outcomes" in out:
                out["expected_outcomes"] = out["outcomes"]

            if isinstance(out.get("notes"), str):
                out["notes"] = [out["notes"]]

            for key in ("expected_outcomes", "new_tasks", "notes"):
                if key in out:
                    out[key] = _ensure_list(out[key])

            return out

        def _repair_json(bad_json: str, error: Exception) -> Optional[Dict[str, Any]]:
            repair_system = (
                "You fix JSON to match a target schema. "
                "Return ONLY a valid JSON object."
            )
            repair_user = textwrap.dedent(
                f"""
                The following JSON does NOT match schema {schema.__name__}.
                Fix it so it validates. Keep content concise.

                Schema hint:
                { _schema_hint() }

                Validation errors:
                { str(error) }

                Invalid JSON:
                { bad_json }
                """
            ).strip()
            kwargs = {
                "model": model,
                "input": [
                    {"role": "system", "content": repair_system},
                    {"role": "user", "content": repair_user},
                ],
                "max_output_tokens": max(800, min(max_output_tokens, 2500)),
                "reasoning": {"effort": "low"},
                "text": {"format": {"type": "json_object"}},
            }
            if temperature is not None:
                kwargs["temperature"] = temperature
            try:
                resp = self.client.responses.create(**kwargs)
            except Exception:
                return None
            raw = resp.output_text
            try:
                cleaned = _extract_json_object(raw)
                return json.loads(cleaned)
            except Exception:
                return None

        input_items = _input_items(user)
        # Try the SDK helper first (best ergonomics).
        if hasattr(self.client.responses, "parse"):
            kwargs: Dict[str, Any] = {
                "model": model,
                "input": input_items,
                "text_format": schema,
                "max_output_tokens": max_output_tokens,
                "reasoning": {"effort": reasoning_effort},
            }
            if temperature is not None:
                kwargs["temperature"] = temperature
            try:
                resp = self.client.responses.parse(**kwargs)
                return resp.output_parsed
            except Exception as e:
                print_warn(
                    "Structured parse failed ({err}); retrying with JSON mode.".format(
                        err=type(e).__name__
                    )
                )

        # Fallback: enforce JSON with instructions and parse ourselves
        retry_user = user + "\n\nReturn ONLY a valid JSON object. If needed, reduce list sizes."
        retries = 2
        last_error: Optional[Exception] = None
        current_max = max_output_tokens
        current_user = retry_user
        for attempt in range(retries):
            kwargs = {
                "model": model,
                "input": _input_items(current_user),
                "max_output_tokens": current_max,
                "reasoning": {"effort": reasoning_effort},
                "text": {"format": {"type": "json_object"}},
            }
            if temperature is not None:
                kwargs["temperature"] = temperature
            resp = self.client.responses.create(**kwargs)
            raw = resp.output_text
            try:
                cleaned = _extract_json_object(raw)
                data = json.loads(cleaned)
                data = _coerce_for_schema(data)
                return schema.model_validate(data)
            except json.JSONDecodeError as e:
                last_error = e
            except ValidationError as e:
                repaired = _repair_json(cleaned, e)
                if repaired is not None:
                    repaired = _coerce_for_schema(repaired)
                    try:
                        return schema.model_validate(repaired)
                    except ValidationError as ve:
                        last_error = ve
                else:
                    last_error = e

            # One retry with more room and stricter reminder.
            current_max = max(current_max, 3000) + 500
            current_user = (
                retry_user
                + "\nKeep responses short: 6-8 items max, 1-2 short sentences per field."
            )

        raw_preview = (raw or "")[:4000]
        raise RuntimeError(f"Failed to parse model JSON after retries.\nRaw:\n{raw_preview}") from last_error

    def text(
        self,
        *,
        model: str,
        reasoning_effort: Literal["none", "low", "medium", "high", "xhigh"] = "medium",
        temperature: Optional[float] = None,
        max_output_tokens: int = 2000,
        system: str,
        user: str,
    ) -> str:
        input_items = [
            {"role": "system", "content": system},
            {"role": "user", "content": user},
        ]
        kwargs: Dict[str, Any] = {
            "model": model,
            "input": input_items,
            "max_output_tokens": max_output_tokens,
            "reasoning": {"effort": reasoning_effort},
        }
        if temperature is not None:
            kwargs["temperature"] = temperature
        resp = self.client.responses.create(**kwargs)
        return resp.output_text


# --- Schemas for constrained agent steps ---
class ClarifyingQuestion(BaseModel):
    id: str
    question: str
    why: str
    answer_type: Literal["free_text", "yes_no", "choice"] = "free_text"
    choices: List[str] = Field(default_factory=list)
    required: bool = True


class ProjectSpec(BaseModel):
    goal: str
    definition_of_done: List[str]
    constraints: List[str] = Field(default_factory=list)
    assumptions: List[str] = Field(default_factory=list)
    clarifying_questions: List[ClarifyingQuestion] = Field(default_factory=list)
    initial_tasks: List[str] = Field(default_factory=list)


Severity = Literal["low", "medium", "high", "critical"]


class Risk(BaseModel):
    id: str
    area: str
    title: str
    description: str
    severity: Severity
    confidence: float = Field(ge=0.0, le=1.0)
    # What would we observe if this risk is real?
    signals: List[str] = Field(default_factory=list)
    # How do we collapse uncertainty?
    resolution_modes: List[Literal["ask_user", "experiment", "write_tests", "design_decision"]] = Field(
        default_factory=list
    )
    user_questions: List[str] = Field(default_factory=list)
    suggested_experiments: List[str] = Field(default_factory=list)


class RiskRegister(BaseModel):
    risks: List[Risk]


class FileWrite(BaseModel):
    path: str
    content: str
    mode: Literal["overwrite", "create_only"] = "overwrite"


class Command(BaseModel):
    cmd: str
    purpose: str
    timeout_sec: int = 300


class StepPlan(BaseModel):
    step_goal: str
    rationale: str
    file_writes: List[FileWrite] = Field(default_factory=list)
    commands: List[Command] = Field(default_factory=list)
    expected_outcomes: List[str] = Field(default_factory=list)
    new_tasks: List[str] = Field(default_factory=list)
    notes: List[str] = Field(default_factory=list)


class FileContent(BaseModel):
    content: str


class DoneCheck(BaseModel):
    done: bool
    rationale: str
    remaining_gaps: List[str] = Field(default_factory=list)
    next_tasks: List[str] = Field(default_factory=list)


# --- Agent state stored on disk ---
@dataclasses.dataclass
class AgentState:
    goal: str
    spec: Optional[Dict[str, Any]] = None
    answers: Dict[str, str] = dataclasses.field(default_factory=dict)
    tasks: List[str] = dataclasses.field(default_factory=list)
    done_criteria: List[str] = dataclasses.field(default_factory=list)
    risks: List[Dict[str, Any]] = dataclasses.field(default_factory=list)
    high_entropy_risk_ids: List[str] = dataclasses.field(default_factory=list)
    history: List[Dict[str, Any]] = dataclasses.field(default_factory=list)

    def to_json(self) -> Dict[str, Any]:
        return dataclasses.asdict(self)

    @staticmethod
    def from_json(data: Dict[str, Any]) -> "AgentState":
        return AgentState(**data)


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


# --- The agent ---
class EntropyAwareAgent:
    def __init__(
        self,
        *,
        goal: str,
        workspace: Path,
        auto: bool,
        model_main: str,
        model_scan: str,
    ) -> None:
        self.goal = goal
        self.workspace = workspace
        self.auto = auto
        self.backend = OpenAIBackend()
        self.policy = CommandPolicy()
        self._llm_call_id = 0

        # Two configs:
        # - main: higher reasoning, no temperature
        # - scan: reasoning none + temperature for diversity (entropy proxy)
        self.model_main = model_main
        self.model_scan = model_scan

        self.state_path = self.workspace / ".entropy_agent" / "state.json"
        self.log_path = self.workspace / ".entropy_agent" / "worklog.md"
        self.workspace.mkdir(parents=True, exist_ok=True)
        (self.workspace / ".entropy_agent").mkdir(parents=True, exist_ok=True)

        if self.state_path.exists():
            self.state = AgentState.from_json(json.loads(read_text(self.state_path)))
        else:
            self.state = AgentState(goal=goal)

    def save(self) -> None:
        write_text(self.state_path, json.dumps(self.state.to_json(), indent=2))
        if not self.log_path.exists():
            write_text(self.log_path, f"# Worklog\n\n## Goal\n{self.goal}\n\n")

    def log(self, title: str, body: str) -> None:
        ts = time.strftime("%Y-%m-%d %H:%M:%S")
        entry = f"\n## {ts} — {title}\n\n{body}\n"
        with self.log_path.open("a", encoding="utf-8") as f:
            f.write(entry)

    def _llm_parse(self, purpose: str, **kwargs: Any) -> BaseModel:
        self._llm_call_id += 1
        call_id = self._llm_call_id
        model = kwargs.get("model")
        reasoning = kwargs.get("reasoning_effort")
        temperature = kwargs.get("temperature")
        max_output_tokens = kwargs.get("max_output_tokens")
        schema = kwargs.get("schema")
        schema_name = schema.__name__ if schema is not None else "unknown"

        print_info(
            "LLM[{id}] {purpose} — sending (model={model}, schema={schema}, reasoning={reasoning}, "
            "temp={temp}, max_out={max_out})".format(
                id=call_id,
                purpose=purpose,
                model=model,
                schema=schema_name,
                reasoning=reasoning,
                temp=temperature,
                max_out=max_output_tokens,
            )
        )
        started = time.perf_counter()
        try:
            resp = self.backend.parse(**kwargs)
        except Exception as e:
            elapsed = time.perf_counter() - started
            print_err(f"LLM[{call_id}] {purpose} — failed after {elapsed:.1f}s: {e}")
            raise
        elapsed = time.perf_counter() - started
        print_info(f"LLM[{call_id}] {purpose} — completed in {elapsed:.1f}s")
        return resp

    # ---------- LLM prompts ----------
    def _system_planner(self) -> str:
        return textwrap.dedent(
            """
            You are an entropy-aware engineering agent.

            Your job is to keep the task in low-entropy territory by:
            - producing concrete artifacts (DoD, constraints, tasks, tests/experiments)
            - explicitly listing unknowns/assumptions
            - proposing experiments/tests to collapse uncertainty
            - asking the user only when necessary (missing domain facts or preferences)

            Rules:
            - Prefer small steps with verifiable outcomes.
            - When unsure, propose an experiment or a precise question.
            - Do not propose destructive commands.
            - Output must follow the provided schema exactly.
            """
        ).strip()

    def _system_risk(self) -> str:
        return textwrap.dedent(
            """
            You are a risk/unknowns analyst for a large software task.

            Produce a risk register with:
            - severity
            - confidence (0..1) about your understanding
            - how to resolve: ask_user vs experiment vs tests vs design decision
            - suggested experiments must be runnable locally (tests, prototypes, commands)
            - 6-8 risks maximum; keep each field concise
            - signals/user_questions/suggested_experiments lists should be <= 2 items each

            Rules:
            - Focus on essential complexity, hidden constraints, integrations, security, scaling, correctness.
            - Prefer actionable, testable items over generic advice.
            - Output must follow the provided schema exactly.
            """
        ).strip()

    def _system_step(self) -> str:
        return textwrap.dedent(
            """
            You are a careful implementation agent.

            Create ONE step plan that:
            - is as small as possible
            - produces verifiable outcomes (tests/commands)
            - writes only the necessary files
            - proposes safe commands only
            - do NOT include file contents in the plan; set file_writes[*].content to an empty string

            Prefer:
            - adding tests first
            - scaffolding interfaces
            - minimal working increments
            - crisp expected outcomes

            Output must follow the provided schema exactly.
            """
        ).strip()

    def _system_file_content(self) -> str:
        return textwrap.dedent(
            """
            You are generating the full content for a single file.

            Rules:
            - Output ONLY a JSON object: {"content": "..."}.
            - Content must be complete, minimal, and directly implements the step.
            - Keep it concise; avoid extra commentary.
            """
        ).strip()

    # ---------- Bootstrap / Spec ----------
    def collect_spec(self) -> None:
        if self.state.spec is not None:
            return

        user_prompt = textwrap.dedent(
            f"""
            Goal:
            {self.goal}

            Produce:
            - definition_of_done as bullet conditions
            - a small set of constraints/assumptions
            - clarifying questions (only the highest leverage)
            - an initial task list that could be executed in small steps

            Keep it practical for a coding agent working locally.
            """
        ).strip()

        spec = self._llm_parse(
            "Collecting spec",
            model=self.model_main,
            reasoning_effort="medium",
            temperature=None,
            system=self._system_planner(),
            user=user_prompt,
            schema=ProjectSpec,
            max_output_tokens=2500,
        )
        assert isinstance(spec, ProjectSpec)
        self.state.spec = spec.model_dump()
        self.state.tasks = list(spec.initial_tasks)
        self.state.done_criteria = list(spec.definition_of_done)
        self.save()

        # Ask user clarifying questions
        answers: Dict[str, str] = {}
        if spec.clarifying_questions:
            print_panel(
                "Clarifying questions",
                "\n".join([f"- ({q.id}) {q.question}\n  why: {q.why}" for q in spec.clarifying_questions]),
            )

        for q in spec.clarifying_questions:
            if q.id in self.state.answers:
                continue

            if self.auto and not q.required:
                # If running auto, we can skip optional questions.
                continue

            if q.answer_type == "yes_no":
                ans = input(f"\n[{q.id}] {q.question} (y/n): ").strip().lower()
                answers[q.id] = "yes" if ans.startswith("y") else "no"
            elif q.answer_type == "choice" and q.choices:
                print(f"\n[{q.id}] {q.question}")
                for i, ch in enumerate(q.choices, 1):
                    print(f"  {i}. {ch}")
                idx = input("Choose number: ").strip()
                try:
                    answers[q.id] = q.choices[int(idx) - 1]
                except Exception:
                    answers[q.id] = input("Enter choice text: ").strip()
            else:
                answers[q.id] = input(f"\n[{q.id}] {q.question}\n> ").strip()

        self.state.answers.update(answers)
        self.save()

        # Write SPEC.md for human readability
        spec_md = [
            "# SPEC",
            "",
            f"## Goal\n{self.goal}",
            "",
            "## Definition of Done",
            *[f"- {c}" for c in self.state.done_criteria],
            "",
            "## Constraints",
            *[f"- {c}" for c in (spec.constraints or [])],
            "",
            "## Assumptions",
            *[f"- {a}" for a in (spec.assumptions or [])],
            "",
            "## Clarifications (answers)",
            *[f"- {k}: {v}" for k, v in self.state.answers.items()],
            "",
            "## Initial tasks",
            *[f"- {t}" for t in self.state.tasks],
            "",
        ]
        write_text(self.workspace / "SPEC.md", "\n".join(spec_md))
        self.log("Collected spec", "\n".join(spec_md))
        print_info("Wrote SPEC.md and initialized task list.")

    # ---------- Entropy scan ----------
    def build_risk_register(self, samples: int = 3) -> None:
        """
        Generate multiple risk registers (diversity sampling) and compute a simple
        disagreement-based "entropy" score per risk.
        """
        if self.state.spec is None:
            raise RuntimeError("Spec not collected yet.")

        spec_text = read_text(self.workspace / "SPEC.md") if (self.workspace / "SPEC.md").exists() else json.dumps(self.state.spec)

        user_prompt = textwrap.dedent(
            f"""
            Using this spec:

            {spec_text}

            Produce a risk register.
            """
        ).strip()

        registers: List[RiskRegister] = []
        for i in range(samples):
            # For entropy scanning, we want diversity.
            # Use reasoning_effort="none" + temperature.
            rr = self._llm_parse(
                f"Risk register sample {i+1}/{samples}",
                model=self.model_scan,
                reasoning_effort="none",
                temperature=0.9,
                system=self._system_risk(),
                user=user_prompt + f"\n\n(Independent sample #{i+1}. Do not copy earlier samples.)",
                schema=RiskRegister,
                max_output_tokens=2500,
            )
            assert isinstance(rr, RiskRegister)
            registers.append(rr)

        # Choose the first register as "primary", then score disagreement across samples.
        primary = registers[0]
        key_counts: Dict[str, int] = {}
        key_to_risks: Dict[str, List[Risk]] = {}

        for rr in registers:
            seen_keys = set()
            for r in rr.risks:
                k = normalize_key(r.title)
                if k in seen_keys:
                    continue
                seen_keys.add(k)
                key_counts[k] = key_counts.get(k, 0) + 1
                key_to_risks.setdefault(k, []).append(r)

        # Compute entropy score for each primary risk
        enriched: List[Dict[str, Any]] = []
        high_entropy_ids: List[str] = []

        for r in primary.risks:
            k = normalize_key(r.title)
            appear_frac = key_counts.get(k, 0) / max(samples, 1)

            # Entropy proxy:
            # - low appearance fraction => disagreement => higher entropy
            # - low confidence => higher entropy
            # - high severity => weight higher
            sev_w = severity_weight(r.severity)
            score = (1.0 - appear_frac) * 0.6 + (1.0 - float(r.confidence)) * 0.4
            score *= (0.5 + 0.5 * sev_w)

            item = r.model_dump()
            item["_entropy"] = {
                "appear_frac": appear_frac,
                "score": score,
                "samples": samples,
            }
            enriched.append(item)

        # Pick "high entropy" ones by threshold + severity
        enriched_sorted = sorted(enriched, key=lambda x: x["_entropy"]["score"], reverse=True)
        for item in enriched_sorted:
            sev = item["severity"]
            score = item["_entropy"]["score"]
            if (sev in ("high", "critical") and score >= 0.35) or (score >= 0.55):
                high_entropy_ids.append(item["id"])

        self.state.risks = enriched_sorted
        self.state.high_entropy_risk_ids = high_entropy_ids
        self.save()

        # Write a human-readable summary
        lines = ["# RISKS", ""]
        for item in enriched_sorted:
            ent = item["_entropy"]
            mark = "🔥" if item["id"] in high_entropy_ids else " "
            lines.append(
                f"- {mark} [{item['severity']}] {item['title']}  "
                f"(confidence={item['confidence']:.2f}, appear={ent['appear_frac']:.2f}, entropy={ent['score']:.2f})"
            )
            lines.append(f"  - area: {item['area']}")
            lines.append(f"  - {item['description']}")
            if item.get("user_questions"):
                for q in item["user_questions"]:
                    lines.append(f"  - ask_user: {q}")
            if item.get("suggested_experiments"):
                for ex in item["suggested_experiments"]:
                    lines.append(f"  - experiment: {ex}")
            lines.append("")
        write_text(self.workspace / "RISKS.md", "\n".join(lines))
        self.log("Built risk register", "\n".join(lines))
        print_info("Wrote RISKS.md and marked high-entropy risks (🔥).")

    # ---------- Next-step selection ----------
    def pick_next_focus(self) -> Tuple[str, Optional[Dict[str, Any]]]:
        """
        Priority:
        1) high-entropy risks (resolve via user question or experiment)
        2) remaining tasks
        """
        # High entropy risks first
        risk_map = {r["id"]: r for r in self.state.risks}
        for rid in self.state.high_entropy_risk_ids:
            r = risk_map.get(rid)
            if not r:
                continue
            # If user questions exist and not answered in freeform notes, ask now.
            if r.get("user_questions"):
                return "ask_user_about_risk", r
            if r.get("suggested_experiments"):
                return "experiment_for_risk", r

        # Otherwise: tasks
        if self.state.tasks:
            return "implement_task", {"task": self.state.tasks[0]}
        return "done_check", None

    # ---------- Human-in-loop ----------
    def ask_user_about_risk(self, risk: Dict[str, Any]) -> None:
        qs = risk.get("user_questions") or []
        body = "\n".join([f"- {q}" for q in qs])
        print_panel(f"High-entropy risk needs input: {risk['title']}", body)
        answers = []
        for i, q in enumerate(qs, 1):
            ans = input(f"\n({i}/{len(qs)}) {q}\n> ").strip()
            answers.append({"q": q, "a": ans})

        self.state.history.append({"type": "risk_user_answers", "risk_id": risk["id"], "answers": answers})
        self.log(f"User input for risk {risk['id']}: {risk['title']}", json.dumps(answers, indent=2))
        # Once we've asked, de-prioritize it (remove from high-entropy list)
        self.state.high_entropy_risk_ids = [x for x in self.state.high_entropy_risk_ids if x != risk["id"]]
        self.save()

    # ---------- Experiments ----------
    def plan_experiment_for_risk(self, risk: Dict[str, Any]) -> StepPlan:
        spec_text = read_text(self.workspace / "SPEC.md") if (self.workspace / "SPEC.md").exists() else ""
        risks_text = read_text(self.workspace / "RISKS.md") if (self.workspace / "RISKS.md").exists() else ""

        prompt = textwrap.dedent(
            f"""
            SPEC:
            {spec_text}

            RISK (high entropy):
            {json.dumps(risk, indent=2)}

            Existing RISKS.md (for context):
            {risks_text}

            Create ONE small experiment step to reduce uncertainty about this risk.
            Prefer: adding a focused test, a minimal prototype, or a quick benchmark.
            """
        ).strip()

        step = self._llm_parse(
            f"Planning experiment for risk {risk.get('id', '?')}: {risk.get('title', '')}".strip(),
            model=self.model_main,
            reasoning_effort="medium",
            temperature=None,
            system=self._system_step(),
            user=prompt,
            schema=StepPlan,
            max_output_tokens=2500,
        )
        assert isinstance(step, StepPlan)
        return step

    def _needs_file_content(self, fw: FileWrite) -> bool:
        if fw.content is None:
            return True
        trimmed = fw.content.strip()
        return trimmed == "" or trimmed.upper() in ("TBD", "TODO")

    def generate_file_content(self, step: StepPlan, fw: FileWrite) -> str:
        spec_text = read_text(self.workspace / "SPEC.md") if (self.workspace / "SPEC.md").exists() else ""
        risks_text = read_text(self.workspace / "RISKS.md") if (self.workspace / "RISKS.md").exists() else ""

        prompt = textwrap.dedent(
            f"""
            SPEC:
            {spec_text}

            RISKS (optional context):
            {risks_text}

            STEP GOAL:
            {step.step_goal}

            RATIONALE:
            {step.rationale}

            EXPECTED OUTCOMES:
            {json.dumps(step.expected_outcomes, indent=2)}

            NOTES:
            {json.dumps(step.notes, indent=2)}

            Write content for this file:
            {fw.path}
            """
        ).strip()

        fc = self._llm_parse(
            f"Writing file content: {fw.path}",
            model=self.model_main,
            reasoning_effort="medium",
            temperature=None,
            system=self._system_file_content(),
            user=prompt,
            schema=FileContent,
            max_output_tokens=2500,
        )
        assert isinstance(fc, FileContent)
        return fc.content

    def execute_step(self, step: StepPlan) -> bool:
        """
        Apply file writes, run commands. Return True if all commands succeed.
        """
        print_panel("Step goal", f"{step.step_goal}\n\n{step.rationale}")

        # Fill file contents in a separate pass (two-stage plan).
        for fw in step.file_writes:
            if fw.content is not None and fw.content.strip():
                fw.content = ""
            if self._needs_file_content(fw):
                fw.content = self.generate_file_content(step, fw)

        # Apply file writes
        for fw in step.file_writes:
            target = self.workspace / fw.path
            if fw.mode == "create_only" and target.exists():
                print_warn(f"Skipping create_only for existing file: {fw.path}")
                continue

            old = read_text(target) if target.exists() else ""
            if old != fw.content:
                diff = unified_diff(old, fw.content, fw.path)
                if diff.strip():
                    print_panel(f"Diff for {fw.path}", diff[:4000] + ("\n... (truncated)" if len(diff) > 4000 else ""))
                if not self.auto:
                    ans = input(f"Apply changes to {fw.path}? (y/n): ").strip().lower()
                    if not ans.startswith("y"):
                        print_warn(f"User declined writing {fw.path}")
                        continue
            write_text(target, fw.content)
            print_info(f"Wrote {fw.path}")

        # Run commands
        all_ok = True
        cmd_outputs: List[Dict[str, Any]] = []

        for c in step.commands:
            safe, why = self.policy.is_safe(c.cmd)
            if not safe and not self.auto:
                print_warn(f"Command blocked by policy: {c.cmd}\nReason: {why}")
                ans = input("Run anyway? (y/n): ").strip().lower()
                if not ans.startswith("y"):
                    cmd_outputs.append({"cmd": c.cmd, "skipped": True, "reason": why})
                    all_ok = False
                    continue
            elif not safe and self.auto:
                print_warn(f"Auto-mode: skipping unsafe command: {c.cmd} ({why})")
                cmd_outputs.append({"cmd": c.cmd, "skipped": True, "reason": why})
                all_ok = False
                continue

            print_info(f"Running: {c.cmd}  ({c.purpose})")
            try:
                code, out = run_command(c.cmd, cwd=self.workspace, timeout_sec=c.timeout_sec)
            except subprocess.TimeoutExpired:
                code, out = 124, f"TIMEOUT after {c.timeout_sec}s"

            print_panel(f"Command output: {c.cmd} (exit {code})", out[:4000] + ("\n... (truncated)" if len(out) > 4000 else ""))
            cmd_outputs.append({"cmd": c.cmd, "exit": code, "output": out})
            if code != 0:
                all_ok = False

        # Update state
        self.state.history.append({"type": "step", "step": step.model_dump(), "cmd_outputs": cmd_outputs})
        self.log(f"Executed step: {step.step_goal}", json.dumps({"step": step.model_dump(), "cmd_outputs": cmd_outputs}, indent=2))

        # Add any newly suggested tasks
        for t in step.new_tasks:
            if t and t not in self.state.tasks:
                self.state.tasks.append(t)

        # If this step was implementing the first task, pop it only if commands succeeded (or no commands).
        if self.state.tasks and step.step_goal.strip().lower().startswith(self.state.tasks[0].strip().lower()[:20].lower()):
            if all_ok:
                self.state.tasks.pop(0)

        self.save()
        return all_ok

    def repair_after_failure(self, failed_step: StepPlan) -> Optional[StepPlan]:
        # Extract the last command outputs from history
        last = None
        for item in reversed(self.state.history):
            if item.get("type") == "step":
                last = item
                break
        if not last:
            return None

        outputs = last.get("cmd_outputs", [])
        spec_text = read_text(self.workspace / "SPEC.md") if (self.workspace / "SPEC.md").exists() else ""

        prompt = textwrap.dedent(
            f"""
            SPEC:
            {spec_text}

            The previous step failed. Here was the step plan:
            {json.dumps(failed_step.model_dump(), indent=2)}

            Here are the command outputs/errors:
            {json.dumps(outputs, indent=2)}

            Create ONE minimal repair step.
            Rules:
            - smallest change that fixes the failing checks
            - prefer adding/adjusting tests only if required
            - safe commands only
            """
        ).strip()

        try:
            repair = self._llm_parse(
                "Planning repair step",
                model=self.model_main,
                reasoning_effort="medium",
                temperature=None,
                system=self._system_step(),
                user=prompt,
                schema=StepPlan,
                max_output_tokens=2500,
            )
            assert isinstance(repair, StepPlan)
            return repair
        except Exception as e:
            print_err(f"Repair planning failed: {e}")
            return None

    # ---------- Done check ----------
    def check_done(self) -> DoneCheck:
        spec_text = read_text(self.workspace / "SPEC.md") if (self.workspace / "SPEC.md").exists() else ""
        remaining_tasks = self.state.tasks[:20]
        last_history = self.state.history[-3:] if len(self.state.history) >= 3 else self.state.history

        prompt = textwrap.dedent(
            f"""
            SPEC:
            {spec_text}

            Remaining tasks (front of queue):
            {json.dumps(remaining_tasks, indent=2)}

            Recent activity:
            {json.dumps(last_history, indent=2)}

            Decide if the project is DONE according to Definition of Done.
            If not done, list the most important remaining gaps and the next tasks.
            """
        ).strip()

        dc = self._llm_parse(
            "Done check",
            model=self.model_main,
            reasoning_effort="medium",
            temperature=None,
            system=self._system_planner(),
            user=prompt,
            schema=DoneCheck,
            max_output_tokens=1500,
        )
        assert isinstance(dc, DoneCheck)
        return dc

    # ---------- Main loop ----------
    def run(self, max_iters: int = 50, max_repairs_per_step: int = 3) -> None:
        self.collect_spec()
        self.build_risk_register(samples=3)

        for it in range(max_iters):
            print_info(f"Iteration {it+1}/{max_iters}")
            kind, payload = self.pick_next_focus()

            if kind == "ask_user_about_risk":
                assert payload is not None
                self.ask_user_about_risk(payload)
                continue

            if kind == "experiment_for_risk":
                assert payload is not None
                risk = payload
                step = self.plan_experiment_for_risk(risk)
                ok = self.execute_step(step)
                repairs = 0
                while not ok and repairs < max_repairs_per_step:
                    repairs += 1
                    print_warn(f"Experiment step failed; attempting repair {repairs}/{max_repairs_per_step}")
                    rep = self.repair_after_failure(step)
                    if not rep:
                        break
                    step = rep
                    ok = self.execute_step(step)
                # De-prioritize this risk after attempting an experiment
                rid = risk["id"]
                self.state.high_entropy_risk_ids = [x for x in self.state.high_entropy_risk_ids if x != rid]
                self.save()
                continue

            if kind == "implement_task":
                assert payload is not None
                task = payload["task"]
                spec_text = read_text(self.workspace / "SPEC.md") if (self.workspace / "SPEC.md").exists() else ""
                risks_text = read_text(self.workspace / "RISKS.md") if (self.workspace / "RISKS.md").exists() else ""

                prompt = textwrap.dedent(
                    f"""
                    SPEC:
                    {spec_text}

                    RISKS:
                    {risks_text}

                    Implement this task in ONE small step:
                    {task}

                    If task is too big, do the smallest useful sub-slice and add follow-up tasks.
                    """
                ).strip()

                task_preview = (task[:80] + "…") if len(task) > 80 else task
                step = self._llm_parse(
                    f"Planning step for task: {task_preview}",
                    model=self.model_main,
                    reasoning_effort="medium",
                    temperature=None,
                    system=self._system_step(),
                    user=prompt,
                    schema=StepPlan,
                    max_output_tokens=3000,
                )
                assert isinstance(step, StepPlan)

                ok = self.execute_step(step)
                repairs = 0
                while not ok and repairs < max_repairs_per_step:
                    repairs += 1
                    print_warn(f"Implementation step failed; attempting repair {repairs}/{max_repairs_per_step}")
                    rep = self.repair_after_failure(step)
                    if not rep:
                        break
                    step = rep
                    ok = self.execute_step(step)

                # If still failing, force human intervention (high-entropy / missing context)
                if not ok:
                    print_panel(
                        "Needs human intervention",
                        "Repairs did not converge. This is likely a high-entropy spot.\n"
                        "Check logs in .entropy_agent/worklog.md and consider adding constraints/tests or answering missing questions.",
                    )
                    if not self.auto:
                        input("Press Enter to continue...")
                continue

            # Done check
            dc = self.check_done()
            print_panel("Done check", f"done={dc.done}\n\n{dc.rationale}\n\nRemaining gaps:\n- " + "\n- ".join(dc.remaining_gaps))
            self.log("Done check", json.dumps(dc.model_dump(), indent=2))
            if dc.done:
                print_info("✅ Task appears complete per Definition of Done.")
                return
            # Add suggested next tasks if they aren't present
            for t in dc.next_tasks:
                if t and t not in self.state.tasks:
                    self.state.tasks.append(t)
            self.save()

        print_warn("Reached max_iters without declaring done. See SPEC.md / RISKS.md / .entropy_agent/worklog.md.")


def main() -> None:
    parser = argparse.ArgumentParser(description="Entropy-aware coding agent (human-in-the-loop + experiments).")
    parser.add_argument("--goal", required=True, help="Big objective/question to work on.")
    parser.add_argument("--workspace", required=True, help="Directory to write files into.")
    parser.add_argument("--auto", action="store_true", help="Auto-run safe steps/commands without asking.")
    parser.add_argument("--max-iters", type=int, default=50, help="Maximum agent loop iterations.")

    # Models:
    # main: used for planning/implementation (higher reasoning)
    # scan: used for entropy scans (reasoning none + temperature)
    parser.add_argument("--model-main", default="gpt-5.2", help="Model for main work steps.")
    parser.add_argument("--model-scan", default="gpt-5.2", help="Model for risk/entropy sampling steps.")
    args = parser.parse_args()

    agent = EntropyAwareAgent(
        goal=args.goal,
        workspace=Path(args.workspace),
        auto=args.auto,
        model_main=args.model_main,
        model_scan=args.model_scan,
    )
    agent.run(max_iters=args.max_iters)


if __name__ == "__main__":
    main()
