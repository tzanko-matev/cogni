from __future__ import annotations

import argparse
from pathlib import Path

from .agent import EntropyAwareAgent


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
