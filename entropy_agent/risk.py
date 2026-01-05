from __future__ import annotations

from typing import Any, Dict, List, Optional

from .models import RiskRegister
from .utils import normalize_key, severity_weight


def enrich_risks(
    registers: List[RiskRegister],
    samples: int,
    *,
    appear_frac_by_primary_id: Optional[Dict[str, float]] = None,
) -> List[Dict[str, Any]]:
    if not registers:
        return []

    primary = registers[0]
    key_counts: Dict[str, int] = {}

    for rr in registers:
        seen_keys = set()
        for r in rr.risks:
            k = normalize_key(r.title)
            if k in seen_keys:
                continue
            seen_keys.add(k)
            key_counts[k] = key_counts.get(k, 0) + 1

    enriched: List[Dict[str, Any]] = []
    for r in primary.risks:
        if appear_frac_by_primary_id and r.id in appear_frac_by_primary_id:
            appear_frac = appear_frac_by_primary_id[r.id]
        else:
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

    return sorted(enriched, key=lambda x: x.get("_entropy", {}).get("score", 0.0), reverse=True)


def select_high_entropy_ids(risks: List[Dict[str, Any]]) -> List[str]:
    high_entropy_ids: List[str] = []
    for item in risks:
        ent = item.get("_entropy") or {}
        score = ent.get("score")
        if score is None:
            continue
        sev = item.get("severity")
        if (sev in ("high", "critical") and score >= 0.35) or (score >= 0.55):
            high_entropy_ids.append(item["id"])
    return high_entropy_ids


def merge_risks(
    existing: List[Dict[str, Any]],
    incoming: List[Dict[str, Any]],
    *,
    iteration: Optional[int] = None,
) -> List[Dict[str, Any]]:
    incoming_by_key = {normalize_key(r["title"]): r for r in incoming if r.get("title")}
    merged: List[Dict[str, Any]] = []

    for current in existing:
        key = normalize_key(current.get("title", ""))
        if key and key in incoming_by_key:
            inc = incoming_by_key.pop(key)
            updated = dict(current)
            updated.update(inc)
            if current.get("id"):
                updated["id"] = current["id"]
            if iteration is not None:
                updated["_last_seen_iter"] = iteration
            merged.append(updated)
        else:
            merged.append(current)

    for inc in incoming_by_key.values():
        if iteration is not None:
            inc = dict(inc)
            inc["_last_seen_iter"] = iteration
        merged.append(inc)

    return sorted(merged, key=lambda x: x.get("_entropy", {}).get("score", 0.0), reverse=True)
