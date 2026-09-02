#!/usr/bin/env python3
"""Run a deterministic synthesized-command STT smoke benchmark in the speech image."""

from __future__ import annotations

import json
import re

from keelmesh_speech.service import RUNTIME


COMMANDS = (
    "Patrol the beach, stay within one nautical mile from the beach as long as ocean depth permits.",
    "Select Gannet and Osprey, form a wedge, and keep thirty percent battery reserve.",
    "Send Watch Shoal to patrol the shoreline and avoid shallow water.",
)


def words(value: str) -> list[str]:
    return re.findall(r"[a-z0-9]+", value.lower())


def word_error_rate(expected: str, actual: str) -> float:
    reference, hypothesis = words(expected), words(actual)
    row = list(range(len(hypothesis) + 1))
    for index, reference_word in enumerate(reference, 1):
        next_row = [index]
        for other, hypothesis_word in enumerate(hypothesis, 1):
            next_row.append(
                min(next_row[-1] + 1, row[other] + 1, row[other - 1] + (reference_word != hypothesis_word))
            )
        row = next_row
    return row[-1] / max(1, len(reference))


for command in COMMANDS:
    audio = RUNTIME.synthesize(command, "morgan")
    result = RUNTIME.transcribe(audio, ".wav")
    print(
        json.dumps(
            {
                "expected": command,
                "actual": result["text"],
                "word_error_rate": round(word_error_rate(command, result["text"]), 3),
                "latency_ms": result["latency_ms"],
                "duration_seconds": result["duration_seconds"],
                "real_time_factor": result["real_time_factor"],
            }
        )
    )
