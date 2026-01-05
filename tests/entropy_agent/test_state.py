from __future__ import annotations

from entropy_agent.state import AgentState


def test_agent_state_roundtrip() -> None:
    state = AgentState(
        goal="goal",
        spec={"k": "v"},
        answers={"q1": "a1"},
        tasks=["t1"],
        done_criteria=["d1"],
        risks=[{"id": "r1"}],
        high_entropy_risk_ids=["r1"],
        history=[{"type": "step"}],
    )
    restored = AgentState.from_json(state.to_json())
    assert restored == state
