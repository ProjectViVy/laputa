"""Synthetic file-system tests; never reads a real profile."""

import hashlib
import importlib


def test_independent_sources_are_opaque_and_role_bound(tmp_path):
    m = importlib.import_module("garden_hermes.markdown")
    shared, home = tmp_path / "shared", tmp_path / "home"
    role = shared / "personas" / "songbai"
    role.mkdir(parents=True)
    home.mkdir()
    bodies = {
        "MISSION.MD": b"User assigned mission.\r\nNo headings required.",
        "IDENTITY.MD": b"# Any heading\n---\npermission: not authorization\n",
        "RELATIONSHIP.MD": b"Plain prose.",
        "REDLINE.MD": b"Do not disclose private information.",
    }
    for name, body in bodies.items():
        (role / name).write_bytes(body)
    (home / "SOUL.md").write_bytes(b"Adopted synthetic soul")
    scope = m.bind_scope(
        shared_root=shared,
        home=home,
        persona_id="songbai",
        binding_revision="fixture-v1",
        consent_ref="synthetic-consent",
        surface="cli",
        approved_mission_digest=hashlib.sha256(bodies["MISSION.MD"]).hexdigest(),
        managed_soul_digest=hashlib.sha256((home / "SOUL.md").read_bytes()).hexdigest(),
        max_source_bytes=10000,
        max_soul_chars=10000,
    )
    snapshot = m.read_sources(scope)
    assert snapshot.file_bytes["IDENTITY.MD"] == bodies["IDENTITY.MD"]
    assert snapshot.file_bytes["MISSION.MD"] == bodies["MISSION.MD"]
    assert set(snapshot.file_bytes) == set(m.ASSEMBLY_FILES)
    assert scope.source_dir == role.resolve()
    assert snapshot.file_bytes["DREAM.MD"] == b""


import pytest

from garden_hermes.domain import BoundaryError, Conflict
from garden_hermes.markdown import bind_scope, read_sources


@pytest.mark.parametrize(
    "bad_id", ["../other", "other/name", "other\\name", ".", "..", "", None, "x\x00y"]
)
def test_role_identifiers_cannot_escape_or_fall_back(persona, bad_id):
    with pytest.raises(BoundaryError):
        bind_scope(**{**persona.settings, "persona_id": bad_id})


@pytest.mark.parametrize(
    "field,value", [("consent_ref", True), ("binding_revision", 42)]
)
def test_binding_metadata_is_typed_not_truthy(persona, field, value):
    with pytest.raises(BoundaryError):
        bind_scope(**{**persona.settings, field: value})


def test_unused_user_world_and_notes_are_not_read(persona):
    (persona.role / "USER.MD").symlink_to("/not/an/approved/file")
    (persona.role / "WORLD.MD").write_bytes(b"\xff")
    (persona.role / "notes.md").write_bytes(b"\xff")
    assert "USER.MD" not in read_sources(persona.scope).file_bytes


def test_unapproved_mission_change_is_not_propagated(persona):
    (persona.role / "MISSION.MD").write_text("Not the approved mission.")
    with pytest.raises(Conflict, match="mission_version"):
        read_sources(persona.scope)


def test_source_link_to_another_role_is_rejected(persona):
    other = persona.shared / "personas" / "other"
    other.mkdir()
    (other / "IDENTITY.MD").write_text("Other role")
    (persona.role / "IDENTITY.MD").unlink()
    (persona.role / "IDENTITY.MD").symlink_to(other / "IDENTITY.MD")
    with pytest.raises(BoundaryError, match="symlink"):
        read_sources(persona.scope)


def test_native_target_requires_explicit_adoption_version(persona):
    (persona.home / "SOUL.md").write_text("Untracked manual edit")
    with pytest.raises(Conflict, match="soul"):
        read_sources(persona.scope)


def test_stale_role_binding_is_rejected_before_read(persona):
    with pytest.raises(Conflict, match="binding"):
        bind_scope(**{**persona.settings, "binding_probe": lambda: "other-role"})


def test_utf8_soul_budget_is_not_a_single_source_file_byte_budget(persona):
    from garden_hermes.domain import digest

    soul = ("松" * 30).encode("utf-8")
    (persona.home / "SOUL.md").write_bytes(soul)
    scope = bind_scope(
        **{
            **persona.settings,
            "managed_soul_digest": digest(soul),
            "max_source_bytes": 64,
            "max_soul_chars": 100,
        }
    )
    assert read_sources(scope).soul_before == soul
