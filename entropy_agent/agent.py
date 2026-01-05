from __future__ import annotations

import json
import subprocess
import textwrap
import time
from pathlib import Path
from typing import Any, Dict, List, Literal, Optional, Tuple

from pydantic import BaseModel

from .backend import OpenAIBackend
from .console import print_err, print_info, print_panel, print_warn
from .models import (
    ClarifyingQuestion,
    Command,
    DoneCheck,
    FileContent,
    FileWrite,
    ProjectSpec,
    Risk,
    RiskRegister,
    StepPlan,
)
from .policy import CommandPolicy
from .state import AgentState
from .utils import normalize_key, read_text, run_command, severity_weight, unified_diff, write_text


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
