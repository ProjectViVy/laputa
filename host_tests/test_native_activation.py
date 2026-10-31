"""Actual Hermes loop/outbound client boundary; entirely synthetic and offline."""

import copy
from types import SimpleNamespace

import pytest

from garden_hermes.activation import observe_identity
from garden_hermes.apply import apply
from garden_hermes.domain import FileReplacement
from garden_hermes.markdown import read_sources


@pytest.mark.parametrize("truncated", [False, True])
def test_actual_outbound_request_new_session_gets_written_identity(
    persona, monkeypatch, truncated
):
    from hermes_cli.plugins_loader import _plugin_home_scope
    from hermes_state import SessionDB
    from openai.types.chat import ChatCompletion
    from run_agent import AIAgent

    monkeypatch.setenv("HERMES_HOME", str(persona.home))
    import json

    (persona.home / "config.yaml").write_text(
        json.dumps({"model": {"context_length": 128000}}), encoding="utf-8"
    )
    monkeypatch.setattr(
        "agent.title_generator.generate_title",
        lambda *a, **kw: "Synthetic session title",
    )
    packets = []
    db = SessionDB(db_path=persona.home / "state.db")

    def make_agent(session):
        actor = AIAgent(
            model="fixture-chat",
            provider="custom",
            api_key="synthetic-fixture",
            base_url="http://127.0.0.1:9/v1",
            api_mode="chat_completions",
            enabled_toolsets=[],
            quiet_mode=True,
            skip_context_files=True,
            load_soul_identity=True,
            skip_memory=True,
            skip_background_review=True,
            session_db=db,
            session_id=session,
            platform="cli",
        )

        def create(**kwargs):
            from agent.system_prompt import _agent_home

            packets.append(
                (
                    actor.session_id,
                    copy.deepcopy(kwargs["messages"]),
                    _agent_home(actor),
                    bool(actor.ephemeral_system_prompt),
                )
            )
            return ChatCompletion(
                id="fixture-response",
                model="fixture-chat",
                object="chat.completion",
                created=0,
                choices=[
                    {
                        "index": 0,
                        "finish_reason": "stop",
                        "message": {"role": "assistant", "content": "Synthetic reply."},
                    }
                ],
                usage={"prompt_tokens": 2, "completion_tokens": 2, "total_tokens": 4},
            )

        client = SimpleNamespace(
            chat=SimpleNamespace(completions=SimpleNamespace(create=create)),
            close=lambda: None,
        )
        actor._create_request_openai_client = lambda **kw: client
        return actor

    with _plugin_home_scope(persona.home):
        old = make_agent("fixture-session-before")
        result = old.run_conversation("First visible turn.")
        assert not result["failed"], result.get("error")
        snapshot = read_sources(persona.scope)
        candidate = FileReplacement(
            persona.scope.binding_key,
            "IDENTITY.MD",
            snapshot.source_versions["IDENTITY.MD"],
            b"SYNTHETIC_IDENTITY_AFTER_APPLY",
        )
        applied = apply(persona.scope, snapshot, candidate, enter_commit=lambda: True)
        assert applied.state == "persisted_verified"
        expected = persona.scope.soul_path.read_bytes()
        if truncated:
            (persona.home / "config.yaml").write_text(
                json.dumps(
                    {"model": {"context_length": 128000}, "context_file_max_chars": 30}
                ),
                encoding="utf-8",
            )
        result = old.run_conversation("Second turn, unchanged existing session.")
        assert not result["failed"], result.get("error")
        resumed = make_agent("fixture-session-before")
        result = resumed.run_conversation(
            "Resumed original session in a new Agent object.",
            conversation_history=db.get_messages_as_conversation(
                "fixture-session-before"
            ),
        )
        assert not result["failed"], result.get("error")
        fresh = make_agent("fixture-session-after")
        result = fresh.run_conversation("New session visible turn.")
        assert not result["failed"], result.get("error")
    assert len(packets) == 4
    existing = observe_identity(
        persona.scope,
        expected,
        packets[1][1],
        request_home=packets[1][2],
        overlay_active=packets[1][3],
        request_session_id=packets[1][0],
        previous_session_id=packets[0][0],
    )
    assert existing["state"] == "activation_unverified"
    resumed_observation = observe_identity(
        persona.scope,
        expected,
        packets[2][1],
        request_home=packets[2][2],
        overlay_active=packets[2][3],
        request_session_id=packets[2][0],
        previous_session_id=packets[0][0],
    )
    assert resumed_observation["state"] == "activation_unverified"
    observed = observe_identity(
        persona.scope,
        expected,
        packets[3][1],
        request_home=packets[3][2],
        overlay_active=packets[3][3],
        request_session_id=packets[3][0],
        previous_session_id=packets[0][0],
    )
    assert observed["state"] == (
        "activation_unverified" if truncated else "observed_new_session"
    )
    if not truncated:
        assert "SYNTHETIC_IDENTITY_AFTER_APPLY" in packets[3][1][0]["content"]
    assert "SYNTHETIC_IDENTITY_AFTER_APPLY" not in packets[1][1][0]["content"]
    from agent.title_generator import wait_for_title_upgrades

    wait_for_title_upgrades()
    old.release_clients()
    resumed.release_clients()
    fresh.release_clients()
    db.close()
