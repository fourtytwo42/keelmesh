from memorylab.gnss_eval import build_episode, evaluate_policy


def fixture() -> dict:
    return {
        "id": "gnss-test",
        "scenario_seed": 42,
        "state_checksum": "sha256:source",
        "classification": "simulation_non_sensitive",
        "evidence": [
            {"id": "before", "kind": "link", "tick": 35, "summary": "normal"},
            {"id": "spoof", "kind": "pnt", "tick": 52, "summary": "GNSS jumped 650 m; fix was excluded."},
        ],
    }


def test_episode_is_deterministic_and_source_linked() -> None:
    first = build_episode(fixture())
    second = build_episode(fixture())
    assert first == second
    assert first["source_event_ids"] == ["before", "spoof"]
    assert first["artifact_checksum"].startswith("sha256:")


def test_baseline_and_candidate_use_same_episode_seed() -> None:
    episode = build_episode(fixture())
    baseline = evaluate_policy(episode, "baseline-v1")
    candidate = evaluate_policy(episode, "candidate-multisignal-v2")
    assert baseline["detected"] is True
    assert candidate["detected"] is True
    assert baseline["seed"] == candidate["seed"] == 42
