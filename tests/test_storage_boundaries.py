import os
import subprocess
import sys
from dataclasses import replace

import pytest

from garden_hermes.domain import BoundaryError, Conflict
from garden_hermes.markdown import bind_scope, read_sources


@pytest.mark.skipif(not hasattr(os, "mkfifo"), reason="POSIX FIFO boundary")
def test_special_file_is_rejected_without_blocking(tmp_path):
    pipe = tmp_path / "fifo"
    os.mkfifo(pipe)
    code = (
        "from pathlib import Path; from garden_hermes.markdown import safe_read; safe_read(Path("
        + repr(str(pipe))
        + "),100)"
    )
    result = subprocess.run(
        [sys.executable, "-c", code],
        capture_output=True,
        text=True,
        timeout=2,
        check=False,
    )
    assert result.returncode == 1 and "not_an_unaliased_regular_file" in result.stderr


@pytest.mark.parametrize("folder", ["shared", "role", "home"])
def test_world_writable_binding_is_rejected(persona, folder):
    path = getattr(persona, folder)
    path.chmod(0o777)
    with pytest.raises(BoundaryError):
        bind_scope(**persona.settings)


def test_corrupt_recovery_is_not_overwritten_by_next_apply(persona):
    from garden_hermes.apply import apply

    first = apply(
        persona.scope, read_sources(persona.scope), None, enter_commit=lambda: True
    )
    after_scope = replace(persona.scope, managed_soul_digest=first.soul_digest)
    record = (persona.scope.recovery_dir / "APPLY.md").read_bytes()
    (
        persona.scope.recovery_dir / first.operation_id / "before" / "SOUL.md"
    ).write_bytes(b"CORRUPT")
    (persona.role / "IDENTITY.MD").write_bytes(b"Authorized human edit")
    with pytest.raises(Conflict):
        apply(after_scope, read_sources(after_scope), None, enter_commit=lambda: True)
    assert (persona.scope.recovery_dir / "APPLY.md").read_bytes() == record


def test_human_edit_syncs_with_zero_model_calls_and_no_source_rewrite(persona):
    from types import SimpleNamespace

    from garden_hermes.apply import apply
    from garden_hermes.proposal import propose

    first = apply(
        persona.scope, read_sources(persona.scope), None, enter_commit=lambda: True
    )
    scope = replace(persona.scope, managed_soul_digest=first.soul_digest)
    (persona.role / "IDENTITY.MD").write_bytes(b"Human-edited free text.")
    stamp = (persona.role / "IDENTITY.MD").stat().st_mtime_ns
    snapshot = read_sources(scope)

    def forbidden(**kwargs):
        raise AssertionError("Empty evidence must not call a model")

    assert (
        propose(
            scope,
            snapshot,
            [],
            SimpleNamespace(complete_structured=forbidden),
            cancelled=lambda: False,
        ).state
        == "no_change"
    )
    assert (
        apply(scope, snapshot, None, enter_commit=lambda: True).state
        == "persisted_verified"
    )
    assert b"Human-edited free text." in scope.soul_path.read_bytes()
    assert (persona.role / "IDENTITY.MD").stat().st_mtime_ns == stamp
