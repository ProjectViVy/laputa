"""Abrupt process exit between real file publications, not a simulated return value."""

import json
import subprocess
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest

from garden_hermes.apply import reconcile, resume_host
from garden_hermes.domain import Conflict


@pytest.mark.parametrize("point", ["source", "soul"])
def test_process_exit_requires_inspection_and_never_replays_model(persona, point):
    settings = {
        k: str(v) if isinstance(v, Path) else v for k, v in persona.settings.items()
    }
    child = """
import os,json,sys
from pathlib import Path
from garden_hermes.markdown import bind_scope,read_sources
from garden_hermes.domain import FileReplacement
from garden_hermes.apply import apply
scope=bind_scope(**json.loads(sys.argv[1]));snapshot=read_sources(scope)
candidate=FileReplacement(scope.binding_key,'IDENTITY.MD',snapshot.source_versions['IDENTITY.MD'],b'AFTER_ABRUPT_EXIT')
target=scope.source_dir/'IDENTITY.MD' if sys.argv[2]=='source' else scope.soul_path
original=os.replace
def publish(src,dst):
    original(src,dst)
    if Path(dst)==target: os._exit(71)
os.replace=publish
apply(scope,snapshot,candidate,enter_commit=lambda:True)
"""
    result = subprocess.run(
        [sys.executable, "-c", child, json.dumps(settings), point],
        timeout=15,
        check=False,
    )
    assert result.returncode == 71
    marker = (persona.scope.recovery_dir / "APPLY.md").read_bytes()
    observed = reconcile(persona.scope)
    assert observed.state == (
        "observed_partial_unconfirmed"
        if point == "source"
        else "applied_bytes_unconfirmed"
    )
    with pytest.raises(Conflict):
        resume_host(persona.scope, enter_commit=lambda: True)
    from garden_hermes.plugin import PersonaController

    config = {
        "enabled": True,
        "source_access_approved": True,
        "writes_approved": True,
        "model_use_approved": True,
        "provider": "custom",
        "model": "fixture",
        "scope": settings,
    }
    calls = []
    llm = SimpleNamespace(complete_structured=lambda **kw: calls.append(kw))
    sdk = SimpleNamespace(get_config=lambda *a: config, llm=llm)
    controller = PersonaController(sdk, config)
    stage = SimpleNamespace(
        home=persona.home,
        surface="cli",
        items=[],
        cancelled=lambda: False,
        try_enter_commit=lambda: True,
    )
    assert controller.run(stage) == "rejected"
    assert not calls
    assert (persona.scope.recovery_dir / "APPLY.md").read_bytes() == marker
