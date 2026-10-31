"""Short, cooperative single-runtime commit; two files are explicitly not ACID."""

import threading
import uuid
from contextlib import ExitStack

from .assemble import assemble
from .domain import ApplyResult, Cancelled, Conflict, digest
from .markdown import atomic_replace, prepared_replace, read_sources, safe_read
from .recovery import (
    prepared_materials,
    prune_other_materials,
    read_receipt,
    write_receipt,
)

_COMMIT_LOCK = threading.RLock()


def apply(scope, snapshot, candidate=None, *, enter_commit):
    target = assemble(scope, snapshot, candidate)
    if target == snapshot.soul_before and (
        candidate is None
        or candidate.replacement == snapshot.file_bytes[candidate.file_key]
    ):
        return ApplyResult("", "no_change", soul_digest=digest(target))
    operation_id = uuid.uuid4().hex
    with ExitStack() as stack:
        record = stack.enter_context(
            prepared_materials(scope, snapshot, candidate, target, operation_id)
        )
        publish_soul = stack.enter_context(prepared_replace(scope.soul_path, target))
        publish_source = None
        if (
            candidate is not None
            and candidate.replacement != snapshot.file_bytes[candidate.file_key]
        ):
            publish_source = stack.enter_context(
                prepared_replace(
                    scope.source_dir / candidate.file_key, candidate.replacement
                )
            )
        with _COMMIT_LOCK:
            scope.assert_current()
            previous = read_receipt(scope)
            if previous:
                from .recovery import validate_materials

                validate_materials(scope, previous)
            if previous and previous["state"] not in ("verified", "rolled_back"):
                raise Conflict("unresolved_last_apply")
            if read_sources(scope) != snapshot:
                raise Conflict("source_or_host_drift")
            if not enter_commit():
                raise Cancelled("cancelled_before_commit")
            write_receipt(scope, record, "prepared")
            source_persisted = False
            try:
                if (
                    candidate is not None
                    and candidate.replacement != snapshot.file_bytes[candidate.file_key]
                ):
                    publish_source()
                    source_persisted = True
                publish_soul()
            except OSError:
                from .recovery import inspect_state

                if (
                    source_persisted
                    and inspect_state(scope, record) == "source_saved_host_pending"
                ):
                    try:
                        write_receipt(scope, record, "source_pending")
                    except OSError:
                        return ApplyResult(
                            operation_id,
                            "unknown_outcome",
                            reason="receipt_write_failed",
                        )
                    return ApplyResult(
                        operation_id,
                        "source_saved_host_pending",
                        digest(candidate.replacement),
                        digest(snapshot.soul_before),
                    )
                return ApplyResult(
                    operation_id,
                    "unknown_outcome",
                    reason="io_failure_requires_inspection",
                )
            if (
                candidate is not None
                and safe_read(
                    scope.source_dir / candidate.file_key, scope.max_source_bytes
                )[0]
                != candidate.replacement
            ):
                raise Conflict("source_readback_mismatch")
            if safe_read(scope.soul_path, 4 * scope.max_soul_chars)[0] != target:
                raise Conflict("soul_readback_mismatch")
            try:
                write_receipt(scope, record, "verified")
            except OSError:
                return ApplyResult(
                    operation_id, "unknown_outcome", reason="receipt_write_failed"
                )
        prune_other_materials(scope, operation_id)
        return ApplyResult(
            operation_id,
            "persisted_verified",
            digest(candidate.replacement) if candidate else None,
            digest(target),
        )


def reconcile(scope):
    """Read-only inspection; never retry an ambiguous source write."""
    from .recovery import inspect_state

    record = read_receipt(scope)
    if record is None:
        return ApplyResult("", "no_recovery")
    state = inspect_state(scope, record)
    if state == "source_saved_host_pending" and record["state"] != "source_pending":
        state = "observed_partial_unconfirmed"
    return ApplyResult(
        record["operation_id"],
        state,
        record["source_after"]
        if state in ("persisted_verified", "source_saved_host_pending")
        else None,
        record["soul_after"] if state == "persisted_verified" else None,
    )


def resume_host(scope, *, enter_commit):
    """Only resume a recorded durable-source/unchanged-host partial commit."""
    from .recovery import inspect_state, saved_body

    with _COMMIT_LOCK:
        record = read_receipt(scope)
        if record is None:
            raise Conflict("no_recovery")
        state = inspect_state(scope, record)
        if state == "persisted_verified":
            return reconcile(scope)
        if state != "source_saved_host_pending" or record["state"] != "source_pending":
            raise Conflict("inspect_only_unknown_outcome")
        target = saved_body(scope, record, "after", "SOUL.md")
        if not enter_commit():
            raise Cancelled("cancelled_before_commit")
        try:
            atomic_replace(scope.soul_path, target)
        except OSError:
            return ApplyResult(
                record["operation_id"], "unknown_outcome", reason="resume_io_failure"
            )
        if safe_read(scope.soul_path, 4 * scope.max_soul_chars)[0] != target:
            raise Conflict("soul_readback_mismatch")
        write_receipt(scope, record, "verified")
        return reconcile(scope)


def rollback(scope, *, authorized_operation_id, enter_commit):
    """Trusted administrative entry, never exposed to a model.

    Caller must authenticate fresh owner approval; an operation ID is NOT a credential.
    Refuse all subsequent source/host edits instead of overwriting them.
    """
    import os

    from .recovery import inspect_state, saved_body

    record = read_receipt(scope)
    if (
        record is None
        or not isinstance(authorized_operation_id, str)
        or record["operation_id"] != authorized_operation_id
    ):
        raise Conflict("explicit_operation_authorization_required")
    key = record["source_file"]
    before_soul = saved_body(scope, record, "before", "SOUL.md")
    before_source = saved_body(scope, record, "before", key) if key != "-" else None
    with ExitStack() as stack:
        publish_soul = stack.enter_context(
            prepared_replace(scope.soul_path, before_soul)
        )
        publish_source = None
        if key != "-" and record["version." + key] != "-":
            publish_source = stack.enter_context(
                prepared_replace(scope.source_dir / key, before_source)
            )
        with _COMMIT_LOCK:
            if read_receipt(scope) != record:
                raise Conflict("stale_recovery_operation")
            if inspect_state(scope, record) not in (
                "persisted_verified",
                "applied_bytes_unconfirmed",
            ):
                raise Conflict("rollback_requires_exact_after_versions")
            if not enter_commit():
                raise Cancelled("cancelled_before_rollback")
            try:
                if key != "-":
                    if publish_source is not None:
                        publish_source()
                    else:
                        (scope.source_dir / key).unlink()
                        fd = os.open(
                            scope.source_dir,
                            os.O_RDONLY | getattr(os, "O_DIRECTORY", 0),
                        )
                        try:
                            os.fsync(fd)
                        finally:
                            os.close(fd)
                publish_soul()
                if (
                    safe_read(scope.soul_path, 4 * scope.max_soul_chars)[0]
                    != before_soul
                ):
                    raise Conflict("rollback_readback_mismatch")
                if (
                    key != "-"
                    and record["version." + key] != "-"
                    and safe_read(scope.source_dir / key, scope.max_source_bytes)[0]
                    != before_source
                ):
                    raise Conflict("rollback_source_readback_mismatch")
                write_receipt(scope, record, "rolled_back")
            except OSError:
                return ApplyResult(
                    record["operation_id"],
                    "rollback_unknown",
                    reason="inspect_before_retry",
                )
            return ApplyResult(
                record["operation_id"], "rolled_back", soul_digest=digest(before_soul)
            )
