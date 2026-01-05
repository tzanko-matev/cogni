from __future__ import annotations

from entropy_agent.models import Risk, RiskRegister
from entropy_agent.risk import enrich_risks, merge_risks, select_high_entropy_ids


def test_enrich_risks_scores_disagreement_higher() -> None:
    r1a = Risk(
        id="r1",
        area="a",
        title="Cache invalidation",
        description="d",
        severity="high",
        confidence=0.9,
    )
    r2a = Risk(
        id="r2",
        area="a",
        title="Timeouts",
        description="d",
        severity="high",
        confidence=0.9,
    )
    r1b = Risk(
        id="r1b",
        area="a",
        title="Cache invalidation",
        description="d",
        severity="high",
        confidence=0.9,
    )

    reg1 = RiskRegister(risks=[r1a, r2a])
    reg2 = RiskRegister(risks=[r1b])

    enriched = enrich_risks([reg1, reg2], samples=2)
    assert enriched[0]["title"] == "Timeouts"
    assert enriched[0]["_entropy"]["appear_frac"] == 0.5


def test_select_high_entropy_ids_uses_thresholds() -> None:
    risks = [
        {"id": "r1", "severity": "high", "_entropy": {"score": 0.4}},
        {"id": "r2", "severity": "low", "_entropy": {"score": 0.6}},
        {"id": "r3", "severity": "medium", "_entropy": {"score": 0.4}},
    ]
    ids = select_high_entropy_ids(risks)
    assert "r1" in ids
    assert "r2" in ids
    assert "r3" not in ids


def test_merge_risks_updates_existing_preserves_id() -> None:
    existing = [
        {
            "id": "r1",
            "title": "Cache invalidation",
            "severity": "low",
            "_entropy": {"score": 0.1},
        }
    ]
    incoming = [
        {
            "id": "new-id",
            "title": "Cache invalidation",
            "severity": "high",
            "_entropy": {"score": 0.9},
        },
        {
            "id": "r2",
            "title": "Timeouts",
            "severity": "medium",
            "_entropy": {"score": 0.5},
        },
    ]

    merged = merge_risks(existing, incoming, iteration=3)
    assert merged[0]["id"] == "r1"
    assert merged[0]["severity"] == "high"
    assert merged[0]["_last_seen_iter"] == 3
    assert any(item["id"] == "r2" for item in merged)
