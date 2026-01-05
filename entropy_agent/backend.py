from __future__ import annotations

import json
import textwrap
from typing import Any, Dict, List, Literal, Optional

from pydantic import BaseModel, ValidationError

from .console import print_warn


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
