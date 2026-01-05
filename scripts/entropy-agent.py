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

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from entropy_agent.cli import main


if __name__ == "__main__":
    main()
