from garden_hermes.domain import FileReplacement
from garden_hermes.markdown import read_sources


def test_single_source_replacement_rebuilds_native_soul_and_preserves_mission(persona):
    from garden_hermes.apply import apply
    from garden_hermes.assemble import assemble

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"New opaque identity.\r\n---\nNo body schema.",
    )
    expected = assemble(persona.scope, snapshot, candidate)
    result = apply(persona.scope, snapshot, candidate, enter_commit=lambda: True)
    assert result.state == "persisted_verified"
    assert (persona.role / "IDENTITY.MD").read_bytes() == candidate.replacement
    assert (persona.role / "MISSION.MD").read_bytes() == persona.bodies["MISSION.MD"]
    assert (persona.home / "SOUL.md").read_bytes() == expected
    assert persona.bodies["MISSION.MD"] in expected
    assert (persona.scope.recovery_dir / "APPLY.md").is_file()
    assert (
        persona.scope.recovery_dir / result.operation_id / "before" / "SOUL.md"
    ).read_bytes() == snapshot.soul_before
    assert (
        persona.scope.recovery_dir / result.operation_id / "after" / "IDENTITY.MD"
    ).read_bytes() == candidate.replacement


def test_source_saved_host_pending_resumes_without_model_or_source_rewrite(
    persona, monkeypatch
):
    import os

    from garden_hermes.apply import apply, reconcile, resume_host

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Updated source.",
    )
    original = os.replace

    def reject_host(src, dst):
        if str(dst) == str(persona.scope.soul_path):
            raise OSError("synthetic host failure")
        return original(src, dst)

    monkeypatch.setattr(os, "replace", reject_host)
    result = apply(persona.scope, snapshot, candidate, enter_commit=lambda: True)
    assert result.state == "source_saved_host_pending"
    assert (persona.role / "IDENTITY.MD").read_bytes() == candidate.replacement
    assert persona.scope.soul_path.read_bytes() == snapshot.soul_before
    assert reconcile(persona.scope).state == "source_saved_host_pending"
    monkeypatch.setattr(os, "replace", original)
    source_stat = (persona.role / "IDENTITY.MD").stat().st_mtime_ns
    resumed = resume_host(persona.scope, enter_commit=lambda: True)
    assert resumed.state == "persisted_verified"
    assert (persona.role / "IDENTITY.MD").stat().st_mtime_ns == source_stat
    assert reconcile(persona.scope).state == "persisted_verified"


def test_files_are_fully_prepared_before_commit_admission(persona):
    from garden_hermes.apply import apply

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Prepared outside the critical section.",
    )

    def enter():
        assert list(persona.role.glob(".garden-*.md"))
        assert list(persona.home.glob(".garden-*.md"))
        return True

    assert (
        apply(persona.scope, snapshot, candidate, enter_commit=enter).state
        == "persisted_verified"
    )
    assert not list(persona.role.glob(".garden-*.md"))
    assert not list(persona.home.glob(".garden-*.md"))


import pytest

from garden_hermes.domain import BoundaryError, Cancelled, Conflict


@pytest.mark.parametrize("later_edit", [False, True])
def test_explicit_rollback_is_version_checked(persona, later_edit):
    from garden_hermes.apply import apply, rollback

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Updated identity.",
    )
    result = apply(persona.scope, snapshot, candidate, enter_commit=lambda: True)
    if later_edit:
        (persona.role / "IDENTITY.MD").write_bytes(b"Later human edit.")
        with pytest.raises(Conflict):
            rollback(
                persona.scope,
                authorized_operation_id=result.operation_id,
                enter_commit=lambda: True,
            )
        assert (persona.role / "IDENTITY.MD").read_bytes() == b"Later human edit."
    else:
        restored = rollback(
            persona.scope,
            authorized_operation_id=result.operation_id,
            enter_commit=lambda: True,
        )
        assert restored.state == "rolled_back"
        assert (persona.role / "IDENTITY.MD").read_bytes() == persona.bodies[
            "IDENTITY.MD"
        ]
        assert persona.scope.soul_path.read_bytes() == snapshot.soul_before


@pytest.mark.parametrize("path", ["IDENTITY.MD", "MISSION.MD", "SOUL.md"])
def test_concurrent_human_edit_is_not_overwritten(persona, path):
    from garden_hermes.apply import apply

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Candidate",
    )
    target = persona.home / path if path == "SOUL.md" else persona.role / path
    target.write_bytes(b"Later human edit")
    with pytest.raises(Conflict):
        apply(persona.scope, snapshot, candidate, enter_commit=lambda: True)
    assert target.read_bytes() == b"Later human edit"


def test_cancelled_commit_leaves_persona_targets_unchanged(persona):
    from garden_hermes.apply import apply

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Candidate",
    )
    with pytest.raises(Cancelled):
        apply(persona.scope, snapshot, candidate, enter_commit=lambda: False)
    assert persona.scope.soul_path.read_bytes() == snapshot.soul_before
    assert (persona.role / "IDENTITY.MD").read_bytes() == persona.bodies["IDENTITY.MD"]


def test_receipt_failure_after_targets_is_unknown_not_success(persona, monkeypatch):
    import garden_hermes.apply as writer

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        persona.scope.binding_key,
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Candidate",
    )
    original = writer.write_receipt

    def failing(scope, record, state):
        if state == "verified":
            raise OSError("synthetic receipt failure")
        return original(scope, record, state)

    monkeypatch.setattr(writer, "write_receipt", failing)
    outcome = writer.apply(
        persona.scope, snapshot, candidate, enter_commit=lambda: True
    )
    assert outcome.state == "unknown_outcome"
    assert writer.reconcile(persona.scope).state == "applied_bytes_unconfirmed"
    with pytest.raises(Conflict):
        writer.resume_host(persona.scope, enter_commit=lambda: True)


def test_cross_role_candidate_is_rejected(persona):
    from garden_hermes.apply import apply

    snapshot = read_sources(persona.scope)
    candidate = FileReplacement(
        "different-role",
        "IDENTITY.MD",
        snapshot.source_versions["IDENTITY.MD"],
        b"Candidate",
    )
    with pytest.raises(Conflict):
        apply(persona.scope, snapshot, candidate, enter_commit=lambda: True)


def test_output_budget_is_checked_before_any_commit(persona):
    from garden_hermes.apply import apply
    from garden_hermes.markdown import bind_scope

    scope = bind_scope(**dict(persona.settings, max_soul_chars=100))
    with pytest.raises(BoundaryError):
        apply(scope, read_sources(scope), None, enter_commit=lambda: True)
    assert (
        persona.scope.soul_path.read_bytes() == b"Explicitly adopted synthetic target."
    )


def test_drift_failure_removes_unpublished_recovery_materials(persona):
    from garden_hermes.apply import apply

    snapshot = read_sources(persona.scope)
    (persona.role / "IDENTITY.MD").write_bytes(b"Human edit")
    with pytest.raises(Conflict):
        apply(persona.scope, snapshot, None, enter_commit=lambda: True)
    assert not list(persona.scope.recovery_dir.iterdir())
