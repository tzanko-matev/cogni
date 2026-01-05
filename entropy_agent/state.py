from __future__ import annotations

import dataclasses
from typing import Any, Dict, List, Optional


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
