from __future__ import annotations

from entropy_agent.agent import EntropyAwareAgent
from entropy_agent.state import AgentState


def _agent_with_state(state: AgentState) -> EntropyAwareAgent:
    agent = EntropyAwareAgent.__new__(EntropyAwareAgent)
    agent.state = state
    return agent


def test_pick_next_focus_prefers_risk_questions() -> None:
    state = AgentState(
        goal="goal",
        risks=[{"id": "r1", "user_questions": ["q"], "suggested_experiments": []}],
        high_entropy_risk_ids=["r1"],
        tasks=["t1"],
    )
    agent = _agent_with_state(state)
    kind, payload = agent.pick_next_focus()
    assert kind == "ask_user_about_risk"
    assert payload["id"] == "r1"


def test_pick_next_focus_prefers_risk_experiments() -> None:
    state = AgentState(
        goal="goal",
        risks=[{"id": "r1", "user_questions": [], "suggested_experiments": ["exp"]}],
        high_entropy_risk_ids=["r1"],
        tasks=["t1"],
    )
    agent = _agent_with_state(state)
    kind, payload = agent.pick_next_focus()
    assert kind == "experiment_for_risk"
    assert payload["id"] == "r1"


def test_pick_next_focus_falls_back_to_task() -> None:
    state = AgentState(goal="goal", risks=[], high_entropy_risk_ids=[], tasks=["t1"])
    agent = _agent_with_state(state)
    kind, payload = agent.pick_next_focus()
    assert kind == "implement_task"
    assert payload["task"] == "t1"


def test_pick_next_focus_done_check_when_empty() -> None:
    state = AgentState(goal="goal", risks=[], high_entropy_risk_ids=[], tasks=[])
    agent = _agent_with_state(state)
    kind, payload = agent.pick_next_focus()
    assert kind == "done_check"
    assert payload is None
