from __future__ import annotations

from typing import List, Literal

from pydantic import BaseModel, Field


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
