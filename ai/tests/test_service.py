from keelmesh_ai.service import (
    ASSERTIONS,
    OPENROUTER_MODELS,
    Circuit,
    MissionOptionsRequest,
    MissionCommandRequest,
    MissionTargetSelectionRequest,
    deterministic_mission_options,
    digest,
    openai_response_text,
    parse_assistant_markdown,
    parse_eval_json,
    parse_geometry_option_id,
    parse_mission_json,
    parse_mission_command,
    parse_model_json,
    parse_target_selection,
)


def test_mission_command_binds_surround_to_live_contact() -> None:
    request = MissionCommandRequest.model_validate({
        "mission_id": "mission-1",
        "intent": "approach and surround Safe Haven",
        "target_ids": ["vessel-1", "vessel-2"],
        "current_formation": "column",
        "constraints": {},
        "surface_contacts": [{
            "id": "surface-16", "boat_id": "NPC-4116", "name": "MT Safe Haven",
            "callsign": "SAFE HAVEN", "class": "tanker", "activity": "anchored",
            "color_name": "aqua", "color": "#63b9b4", "position": [-71.275, 41.285],
            "heading_deg": 0, "speed_mps": 0, "speed_knots": 0, "length_m": 138,
            "draft_m": 8.2, "navigation_state": "at anchor", "route_name": "anchorage",
            "route": [[-71.275, 41.285]], "looping": False,
            "updated_at": "2026-09-03T12:00:00Z",
        }],
    })
    result = parse_mission_command(
        '{"guidance_kind":"orbit_contact","contact_id":"surface-16","contact_behavior":"surround","dynamic_target":true,"formation":"ring","standoff_m":120,"minimum_reserve":0,"maximum_speed_mps":0,"hold_at_end":true,"summary":"Surround the identified tanker."}',
        request,
    )
    assert result["contact_id"] == "surface-16"
    assert result["dynamic_target"] is True
    assert result["formation"] == "ring"


def test_openai_responses_output_text_is_extracted() -> None:
    assert (
        openai_response_text(
            {
                "output": [
                    {
                        "type": "message",
                        "content": [{"type": "output_text", "text": '{"strategies":[]}'}],
                    }
                ]
            }
        )
        == '{"strategies":[]}'
    )


def test_target_selection_requires_complete_group_for_generic_group_request() -> None:
    request = MissionTargetSelectionRequest.model_validate(
        {
            "mission_id": "mission-1",
            "intent": "Move a group 1 nm east and hold position.",
            "groups": [
                {
                    "id": "group-1",
                    "code": "AG",
                    "name": "Amber Guard",
                    "color_name": "amber",
                    "member_ids": ["vessel-1", "vessel-2"],
                    "formation": "column",
                    "available": True,
                }
            ],
            "vessels": [
                {
                    "id": "vessel-1",
                    "name": "Gannet",
                    "callsign": "Gannet",
                    "designation": "KM-214",
                    "class": "Kestrel",
                    "position": [-71.3, 41.4],
                    "reserve": 0.9,
                    "available": True,
                },
                {
                    "id": "vessel-2",
                    "name": "Osprey",
                    "callsign": "Osprey",
                    "designation": "KM-215",
                    "class": "Mariner",
                    "position": [-71.31, 41.4],
                    "reserve": 0.85,
                    "available": True,
                },
            ],
        }
    )
    ids, explanation = parse_target_selection(
        '{"target_ids":["vessel-1","vessel-2"],"explanation":"Best reserve."}', request
    )
    assert ids == ["vessel-1", "vessel-2"]
    assert explanation == "Best reserve."
    try:
        parse_target_selection(
            '{"target_ids":["vessel-1"],"explanation":"Partial group."}', request
        )
    except ValueError:
        pass
    else:
        raise AssertionError("a generic group request must resolve to one complete group")


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


def test_single_vessel_mission_options_reject_fleet_formations() -> None:
    valid = parse_mission_json(
        '{"strategies":[{"id":"one","name":"Close patrol","description":"Depth-safe coastal track","formation":"independent","speed_factor":0.7,"reserve_bias":0.3,"maneuvers":["join corridor","patrol"]},{"id":"two","name":"Reserve patrol","description":"Lower propulsion demand","formation":"independent","speed_factor":0.5,"reserve_bias":0.8,"maneuvers":["economy entry","safe hold"]}]}',
        1,
        "patrol",
    )
    assert all(item["formation"] == "independent" for item in valid)
    try:
        parse_mission_json(
            '{"strategies":[{"id":"one","name":"Wedge","description":"wrong target semantics","formation":"wedge","speed_factor":0.7,"reserve_bias":0.3,"maneuvers":["form","patrol"]},{"id":"two","name":"Line","description":"wrong target semantics","formation":"line_abreast","speed_factor":0.5,"reserve_bias":0.8,"maneuvers":["form","hold"]}]}',
            1,
            "patrol",
        )
    except ValueError:
        pass
    else:
        raise AssertionError("single-vessel advisor must reject fleet formations")
    try:
        parse_mission_json(
            '{"strategies":[{"id":"one","name":"Solo patrol","description":"Depth-safe coastal track","formation":"independent","speed_factor":0.7,"reserve_bias":0.3,"maneuvers":["patrol coast","regroup on completion"]},{"id":"two","name":"Reserve patrol","description":"Lower propulsion demand","formation":"independent","speed_factor":0.5,"reserve_bias":0.8,"maneuvers":["patrol coast","safe hold"]}]}',
            1,
            "patrol",
            0,
        )
    except ValueError:
        pass
    else:
        raise AssertionError("single-vessel advisor must reject fleet-only maneuver prose")


def test_mock_mission_advisor_is_target_aware() -> None:
    request = MissionOptionsRequest.model_validate(
        {
            "schema_version": 2,
            "mission_id": "mission-1",
            "intent": "patrol the shoreline",
            "guidance_kind": "patrol",
            "target_count": 1,
            "targets": [
                {
                    "id": "v1",
                    "name": "Gannet",
                    "class": "Kestrel",
                    "position": [-71.3, 41.4],
                    "reserve": 0.8,
                    "max_speed_mps": 1.8,
                    "pnt_integrity": "trusted",
                    "uncertainty_m": 4,
                    "group_code": "WS",
                    "group_name": "Watch Shoal",
                    "group_color_name": "amber",
                    "communications": "mesh",
                }
            ],
            "constraints": {},
            "environment": {},
            "operating_areas": 1,
            "exclusion_areas": 0,
            "waypoint_count": 7,
            "geometry_source": "intent:coast",
            "formation_current": "column",
        }
    )
    options = deterministic_mission_options(request)
    assert options[0]["name"] == "Close Shoreline Patrol"
    assert {item["formation"] for item in options} == {"independent"}


def test_mission_context_accepts_stationary_anchored_contact() -> None:
    request = MissionOptionsRequest.model_validate(
        {
            "schema_version": 2,
            "mission_id": "mission-anchor-watch",
            "intent": "approach and observe Safe Haven",
            "guidance_kind": "follow_contact",
            "target_count": 1,
            "targets": [
                {
                    "id": "v1",
                    "name": "Gannet",
                    "class": "Kestrel",
                    "position": [-71.3, 41.4],
                    "reserve": 0.8,
                    "max_speed_mps": 3.4,
                    "pnt_integrity": "trusted",
                    "uncertainty_m": 4,
                    "group_code": "WS",
                    "group_name": "Watch Shoal",
                    "group_color_name": "amber",
                    "communications": "mesh",
                }
            ],
            "constraints": {},
            "environment": {},
            "operating_areas": 0,
            "exclusion_areas": 0,
            "waypoint_count": 1,
            "geometry_source": "intent:contact",
            "formation_current": "column",
            "surface_contacts": [
                {
                    "id": "surface-16",
                    "boat_id": "NPC-4116",
                    "name": "MT Safe Haven",
                    "callsign": "SAFE HAVEN",
                    "class": "tanker",
                    "activity": "anchored · simulated weather hold",
                    "color_name": "aqua",
                    "color": "#63b9b4",
                    "position": [-71.275, 41.285],
                    "heading_deg": 0,
                    "speed_mps": 0,
                    "speed_knots": 0,
                    "length_m": 138,
                    "draft_m": 8.2,
                    "navigation_state": "at anchor",
                    "route_name": "Rhode Island Sound anchorage",
                    "route": [[-71.275, 41.285]],
                    "looping": False,
                    "updated_at": "2026-09-03T12:00:00Z",
                }
            ],
        }
    )
    assert request.surface_contacts[0].navigation_state == "at anchor"
    assert request.surface_contacts[0].route == [(-71.275, 41.285)]


def test_strategy_name_maps_to_bounded_operation_name() -> None:
    strategy_name = "Close Shoreline Patrol"
    mission_name = (
        strategy_name if strategy_name.lower().startswith("operation ") else f"Operation {strategy_name}"
    )
    assert mission_name[:64] == "Operation Close Shoreline Patrol"


def test_geometry_selection_is_limited_to_supplied_options() -> None:
    payload = '{"geometry_option_id":"coastal-corridor-03","strategies":[]}'
    assert (
        parse_geometry_option_id(payload, ["coastal-corridor-01", "coastal-corridor-03"])
        == "coastal-corridor-03"
    )
    try:
        parse_geometry_option_id(payload, ["coastal-corridor-01"])
    except ValueError:
        pass
    else:
        raise AssertionError("provider geometry must be selected from the supplied allow-list")


def test_conversational_mission_reply_is_bounded_and_required() -> None:
    payload = '{"assistant_markdown":"I mapped **three** depth-safe options.","geometry_option_id":"","strategies":[]}'
    assert parse_assistant_markdown(payload) == "I mapped **three** depth-safe options."
    try:
        parse_assistant_markdown('{"geometry_option_id":"","strategies":[]}')
    except ValueError:
        pass
    else:
        raise AssertionError("mission provider reply must contain conversational Markdown")
