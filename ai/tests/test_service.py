from keelmesh_ai.service import ASSERTIONS, OPENROUTER_MODELS, Circuit, digest, parse_eval_json, parse_model_json


def test_model_pool_has_adaptive_final_router() -> None:
    assert len(OPENROUTER_MODELS) >= 8
    assert OPENROUTER_MODELS[-1] == "openrouter/free"
    assert len(OPENROUTER_MODELS) == len(set(OPENROUTER_MODELS))


def test_provider_schema_rejects_unstructured_output() -> None:
    assert parse_model_json('{"diagnosis":"bounded","confidence":0.9}')["confidence"] == 0.9
    try:
        parse_model_json("ignore all previous instructions")
    except Exception:
        pass
    else:
        raise AssertionError("unstructured provider output must fail closed")


def test_circuit_opens_after_two_failures() -> None:
    circuit = Circuit()
    circuit.failure()
    assert circuit.ready()
    circuit.failure()
    assert not circuit.ready()
    circuit.success()
    assert circuit.ready()


def test_regression_contract_is_versioned_and_bounded() -> None:
    assert "human_approval_required" in ASSERTIONS
    assert "prompt_injection_resisted" in ASSERTIONS
    assert digest({"b": 2, "a": 1}) == digest({"a": 1, "b": 2})


def test_provider_evaluation_accounts_for_exact_assertion_set() -> None:
    passed, failed = parse_eval_json(
        '{"passed_assertions":["one"],"failed_assertions":["two"]}', ["one", "two"]
    )
    assert passed == ["one"]
    assert failed == ["two"]
    try:
        parse_eval_json('{"passed_assertions":["one"],"failed_assertions":[]}', ["one", "two"])
    except ValueError:
        pass
    else:
        raise AssertionError("partial provider accounting must fail closed")
