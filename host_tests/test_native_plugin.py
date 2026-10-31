"""SDK entry -> actual finalizer -> managed native worker -> SDK structured LLM -> files."""

import json
import threading
from pathlib import Path
from types import SimpleNamespace

import pytest


@pytest.mark.parametrize(
    "variant",
    [
        "proposal",
        "no_change",
        "protected",
        "timeout",
        "cancelled",
        "disposed",
        "malformed",
        "duplicate",
        "rebind",
        "foreign",
    ],
)
def test_enabled_sdk_entry_closes_visible_evidence_loop(persona, monkeypatch, variant):
    from agent import background_review as review
    from agent.turn_finalizer import finalize_turn
    from hermes_cli.plugins import PluginContext, PluginManager, PluginManifest
    from hermes_cli.plugins_loader import _plugin_home_scope
    from openai.types.chat import ChatCompletion
    from tests.agent.test_turn_finalizer_final_response_persistence import FakeAgent

    from garden_hermes.plugin import register

    settings = {
        k: str(v) if isinstance(v, Path) else v for k, v in persona.settings.items()
    }
    config = {
        "enabled": True,
        "source_access_approved": True,
        "writes_approved": True,
        "model_use_approved": True,
        "provider": "custom",
        "model": "fixture-helper",
        "scope": settings,
    }
    profile = {
        "plugins": {
            "entries": {
                "garden-persona": {
                    "enabled": True,
                    "settings": {"persona": config},
                    "llm": {
                        "allow_provider_override": True,
                        "allowed_providers": ["custom"],
                        "allow_model_override": True,
                        "allowed_models": ["fixture-helper"],
                    },
                }
            }
        }
    }
    (persona.home / "config.yaml").write_text(json.dumps(profile), encoding="utf-8")
    monkeypatch.setenv("HERMES_HOME", str(persona.home))
    calls = []

    def backend(**kwargs):
        calls.append(kwargs)
        if variant == "timeout":
            raise TimeoutError("Synthetic timeout, never a real provider call")
        if variant == "cancelled":
            parent._background_review_run.cancel()
        if variant == "disposed":
            handle.dispose()
        if variant == "rebind":
            config["scope"]["binding_revision"] = "new-binding"
            (persona.home / "config.yaml").write_text(
                json.dumps(profile), encoding="utf-8"
            )
        content = kwargs["messages"][-1]["content"]
        text = content if isinstance(content, str) else content[-1]["text"]
        data = json.loads(text)
        value = {
            "result": "proposal",
            "file_key": "IDENTITY.MD",
            "replacement_text": "SYNTHETIC_LEARNED_IDENTITY",
            "source_item_ids": [data["evidence"][0]["item_id"]],
            "rationale": "A stable synthetic preference.",
            "counterevidence": [],
        }
        if variant == "no_change":
            value = {"result": "no_change"}
        if variant == "protected":
            value["file_key"] = "MISSION.MD"
        if variant == "malformed":
            value["grant"] = True
        return ChatCompletion(
            id="fixture-helper",
            model="fixture-helper",
            object="chat.completion",
            created=0,
            choices=[
                {
                    "index": 0,
                    "finish_reason": "stop",
                    "message": {"role": "assistant", "content": json.dumps(value)},
                }
            ],
            usage={"prompt_tokens": 10, "completion_tokens": 10, "total_tokens": 20},
        )

    monkeypatch.setattr("agent.auxiliary_client.call_llm", backend)

    class Fork:
        _session_messages = ()

        def run_conversation(self, **kwargs):
            return {"completed": True}

        def release_clients(self):
            pass

    monkeypatch.setattr(review, "_parent_can_emit_tool_calls", lambda *a: True)
    monkeypatch.setattr(
        review, "build_cache_parity_fork", lambda *a, **kw: (Fork(), None, False)
    )
    monkeypatch.setattr(review, "_review_tool_whitelist", lambda *a: (set(), set()))
    monkeypatch.setattr(review, "_snapshot_review_usage", lambda *a: {})
    monkeypatch.setattr(review, "_record_review_usage_to_parent", lambda *a: None)
    monkeypatch.setattr("hermes_cli.plugins.invoke_hook", lambda *a, **kw: [])
    if variant == "foreign":
        monkeypatch.setattr(
            "hermes_cli.plugin_session.session_surface", lambda: "qqbot"
        )
    parent = FakeAgent()
    parent._session_db = SimpleNamespace(db_path=persona.home / "state.db")
    parent._background_review_lock = threading.Lock()
    parent._background_review_run = None
    parent._background_review_agent = None
    parent.background_review_callback = None
    parent.memory_notifications = "off"
    parent._emit_auxiliary_failure = lambda *a: None

    def spawn(**kwargs):
        token = review.prepare_background_review_run(parent)
        review._run_review_in_thread(
            parent,
            kwargs["messages_snapshot"],
            "native review",
            review_run=token,
            persona_request=kwargs.get("persona_request"),
        )
        assert token.request_done.is_set()

    parent._spawn_background_review = spawn
    with _plugin_home_scope(persona.home):
        manager = PluginManager(scope_key=str(persona.home))
        manager._cli_ref = object()
        ctx = PluginContext(
            PluginManifest(name="garden-persona", key="garden-persona"), manager
        )
        handle = register(ctx)
        assert handle is not None
        try:
            result = finalize_turn(
                parent,
                final_response="Visible synthetic answer.",
                api_call_count=1,
                interrupted=False,
                failed=False,
                messages=[{"role": "user", "content": "HIDDEN_HISTORY_NOT_EGRESS"}],
                conversation_history=[],
                effective_task_id="task",
                turn_id="turn",
                user_message="HIDDEN_HISTORY_NOT_EGRESS",
                original_user_message="Prefer careful evidence.",
                _should_review_memory=True,
                _turn_exit_reason="fallback_prior_turn_content",
            )
            assert not result["failed"]
            expected = {
                "proposal": "persisted_verified",
                "duplicate": "persisted_verified",
                "no_change": "persisted_verified",
                "protected": "needs_user_authorization",
                "timeout": "analysis_failed",
                "cancelled": "cancelled",
                "disposed": "cancelled",
                "malformed": "rejected",
                "rebind": "rejected",
                "foreign": None,
            }[variant]
            assert getattr(parent, "_last_persona_review_status", None) == expected
            assert len(calls) == (0 if variant == "foreign" else 1)
            if variant == "foreign":
                assert (persona.role / "IDENTITY.MD").read_bytes() == persona.bodies[
                    "IDENTITY.MD"
                ]
                return
            sent = json.dumps(calls[0], ensure_ascii=False)
            assert "Prefer careful evidence." in sent
            assert "HIDDEN_HISTORY_NOT_EGRESS" not in sent
            assert "Explicitly adopted synthetic target." not in sent
            assert (
                calls[0]["provider"] == "custom"
                and calls[0]["model"] == "fixture-helper"
            )
            changed = variant in ("proposal", "duplicate")
            assert (persona.role / "IDENTITY.MD").read_bytes() == (
                b"SYNTHETIC_LEARNED_IDENTITY"
                if changed
                else persona.bodies["IDENTITY.MD"]
            )
            if changed:
                assert (
                    b"SYNTHETIC_LEARNED_IDENTITY"
                    in persona.scope.soul_path.read_bytes()
                )
            assert (persona.role / "MISSION.MD").read_bytes() == persona.bodies[
                "MISSION.MD"
            ]
            if variant == "duplicate":
                receipt = (persona.scope.recovery_dir / "APPLY.md").read_bytes()
                finalize_turn(
                    parent,
                    final_response="Visible synthetic answer.",
                    api_call_count=1,
                    interrupted=False,
                    failed=False,
                    messages=[],
                    conversation_history=[],
                    effective_task_id="task",
                    turn_id="turn",
                    user_message="same replay",
                    original_user_message="Prefer careful evidence.",
                    _should_review_memory=True,
                    _turn_exit_reason="fallback_prior_turn_content",
                )
                assert len(calls) == 1
                assert parent._last_persona_review_status == "no_change"
                assert (persona.scope.recovery_dir / "APPLY.md").read_bytes() == receipt
        finally:
            handle.dispose()
