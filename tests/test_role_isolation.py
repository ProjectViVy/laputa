"""Two real synthetic role directories with intentionally identical initial hashes."""

import shutil
from types import SimpleNamespace

import pytest

from garden_hermes.apply import apply, reconcile
from garden_hermes.domain import Conflict, FileReplacement
from garden_hermes.markdown import bind_scope, read_sources


def test_same_named_files_and_hashes_do_not_authorize_cross_role_recovery(
    persona, tmp_path
):
    other_role = persona.shared / "personas" / "other-role"
    shutil.copytree(persona.role, other_role)
    other_home = tmp_path / "other-home"
    other_home.mkdir()
    (other_home / "SOUL.md").write_bytes(persona.scope.soul_path.read_bytes())
    other = bind_scope(
        **dict(persona.settings, home=other_home, persona_id="other-role")
    )
    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Only role A changes.",
    )
    with pytest.raises(Conflict):
        apply(other, read_sources(other), candidate, enter_commit=lambda: True)
    assert (
        apply(persona.scope, snapshot, candidate, enter_commit=lambda: True).state
        == "persisted_verified"
    )
    assert (other_role / "IDENTITY.MD").read_bytes() == persona.bodies["IDENTITY.MD"]
    shutil.copytree(persona.scope.recovery_dir, other.recovery_dir)
    with pytest.raises(Conflict):
        reconcile(other)
    assert other.soul_path.read_bytes() == snapshot.soul_before


def test_optional_private_host_is_local_assembly_only(persona):
    from garden_hermes.proposal import propose

    path = persona.home / "plugin-data" / "garden" / "HOST.md"
    path.parent.mkdir(parents=True)
    private = b"Host-specific synthetic context."
    path.write_bytes(private)
    scope = bind_scope(**dict(persona.settings, include_host=True))
    snapshot = read_sources(scope)
    calls = []

    def complete(**kwargs):
        calls.append(kwargs)
        return SimpleNamespace(
            parsed={"result": "no_change"}, usage=SimpleNamespace(total_tokens=3)
        )

    assert (
        propose(
            scope,
            snapshot,
            [{"item_id": "u", "role": "user", "text": "Visible"}],
            SimpleNamespace(complete_structured=complete),
            cancelled=lambda: False,
        ).state
        == "no_change"
    )
    assert private.decode() not in str(calls)
    result = apply(scope, snapshot, None, enter_commit=lambda: True)
    assert result.state == "persisted_verified"
    assert scope.soul_path.read_bytes().endswith(private)
    assert path.read_bytes() == private
