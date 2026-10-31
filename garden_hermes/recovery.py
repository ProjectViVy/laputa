"""Only the last Markdown intent + exact private before/after bodies; no ledger or queue."""

import re
import shutil
from contextlib import contextmanager

from .domain import ASSEMBLY_FILES, AUTO_FILES, BoundaryError, Conflict, digest
from .markdown import atomic_replace, reject_symlinks, safe_read

_FIELDS = {
    "operation_id",
    "binding",
    "source_file",
    "source_after",
    "soul_before",
    "soul_after",
    "host_version",
    "state",
} | {"version." + k for k in ASSEMBLY_FILES}


def _version(value):
    return value == "-" or re.fullmatch("[0-9a-f]{64}", value) is not None


def read_receipt(scope):
    body, version = safe_read(scope.recovery_dir / "APPLY.md", 8192, optional=True)
    if version is None:
        return None
    lines = body.decode().splitlines()
    if not lines or lines[0] != "# Garden last apply v1":
        raise Conflict("invalid_recovery_record")
    pairs = [line.split(": ", 1) for line in lines[1:] if line]
    if any(len(pair) != 2 for pair in pairs):
        raise Conflict("invalid_recovery_record")
    record = dict(pairs)
    if (
        set(record) != _FIELDS
        or len(record) != len(pairs)
        or re.fullmatch("[0-9a-f]{32}", record["operation_id"]) is None
        or record["binding"] != scope.binding_key
        or record["source_file"] not in AUTO_FILES | {"-"}
        or record["state"]
        not in ("prepared", "verified", "rolled_back", "source_pending", "unknown")
        or any(
            not _version(record[k])
            for k in _FIELDS
            if k.startswith("version.")
            or k in ("source_after", "soul_before", "soul_after", "host_version")
        )
    ):
        raise Conflict("recovery_binding_or_format_conflict")
    return record


def write_receipt(scope, record, state):
    record = dict(record, state=state)
    body = "# Garden last apply v1\n" + "".join(
        f"{k}: {record[k]}\n" for k in sorted(record)
    )
    atomic_replace(scope.recovery_dir / "APPLY.md", body.encode())
    return record


@contextmanager
def prepared_materials(scope, snapshot, candidate, target, operation_id):
    record = prepare_materials(scope, snapshot, candidate, target, operation_id)
    try:
        yield record
    finally:
        try:
            current = read_receipt(scope)
            if current is None or current["operation_id"] != operation_id:
                shutil.rmtree(scope.recovery_dir / operation_id)
        except (OSError, BoundaryError):
            # Keep uncertain recovery, never remove the material an intent may reference.
            pass


def prepare_materials(scope, snapshot, candidate, target, operation_id):
    reject_symlinks(scope.recovery_dir)
    scope.recovery_dir.mkdir(parents=True, mode=0o700, exist_ok=True)
    if scope.recovery_dir.stat().st_mode & 0o077:
        raise BoundaryError("private_recovery_directory_required")
    directory = scope.recovery_dir / operation_id
    directory.mkdir(mode=0o700)
    record = {
        "operation_id": operation_id,
        "binding": scope.binding_key,
        "source_file": candidate.file_key if candidate else "-",
        "source_after": digest(candidate.replacement) if candidate else "-",
        "soul_before": digest(snapshot.soul_before),
        "soul_after": digest(target),
        "host_version": snapshot.host_version or "-",
        "state": "prepared",
    }
    record.update(
        {"version." + k: snapshot.source_versions[k] or "-" for k in ASSEMBLY_FILES}
    )
    try:
        for when, soul in (("before", snapshot.soul_before), ("after", target)):
            folder = directory / when
            folder.mkdir(mode=0o700)
            atomic_replace(folder / "SOUL.md", soul)
            if candidate:
                atomic_replace(
                    folder / candidate.file_key,
                    snapshot.file_bytes[candidate.file_key]
                    if when == "before"
                    else candidate.replacement,
                )
    except Exception:
        shutil.rmtree(directory)
        raise
    return record


def saved_body(scope, record, when, name):
    if (
        when not in ("before", "after")
        or name not in (record["source_file"], "SOUL.md")
        or name == "-"
    ):
        raise BoundaryError("invalid_recovery_target")
    body, _ = safe_read(
        scope.recovery_dir / record["operation_id"] / when / name,
        max(scope.max_source_bytes, 4 * scope.max_soul_chars),
    )
    expected = (
        record["soul_" + when]
        if name == "SOUL.md"
        else record["source_after"]
        if when == "after"
        else record["version." + name]
    )
    if digest(body) != expected and not (expected == "-" and body == b""):
        raise Conflict("corrupt_recovery_snapshot")
    return body


def prune_other_materials(scope, keep):
    # Only private operation directories with the exact UUID namespace are owned here.
    for path in scope.recovery_dir.iterdir():
        if path.name != keep and re.fullmatch("[0-9a-f]{32}", path.name):
            reject_symlinks(path)
            if path.is_dir():
                shutil.rmtree(path)


def validate_materials(scope, record):
    for when in ("before", "after"):
        saved_body(scope, record, when, "SOUL.md")
        if record["source_file"] != "-":
            saved_body(scope, record, when, record["source_file"])


def inspect_state(scope, record):
    scope.assert_current()
    key = record["source_file"]
    if record["binding"] != scope.binding_key:
        raise Conflict("wrong_recovery_binding")
    for when in ("before", "after"):
        saved_body(scope, record, when, "SOUL.md")
        if key != "-":
            saved_body(scope, record, when, key)
    source_before = source_after = True
    for name in ASSEMBLY_FILES:
        from .domain import OPTIONAL_FILES

        _, version = safe_read(
            scope.source_dir / name,
            scope.max_source_bytes,
            optional=name in OPTIONAL_FILES,
        )
        live = version or "-"
        if name == "MISSION.MD" and live != scope.approved_mission_digest:
            raise Conflict("mission_version_not_authorized")
        before = record["version." + name]
        after = record["source_after"] if name == key else before
        if name == key:
            source_before = live == before
            source_after = live == after
        elif live != before:
            raise Conflict("assembly_source_drift")
    if scope.include_host:
        _, host = safe_read(
            scope.home / "plugin-data" / "garden" / "HOST.md",
            scope.max_source_bytes,
            optional=True,
        )
        if (host or "-") != record["host_version"]:
            raise Conflict("host_context_drift")
    _, soul = safe_read(scope.soul_path, 4 * scope.max_soul_chars)
    if soul == record["soul_after"] and source_after:
        return (
            "persisted_verified"
            if record["state"] == "verified"
            else "applied_bytes_unconfirmed"
        )
    if soul == record["soul_before"] and source_before:
        return "rolled_back" if record["state"] == "rolled_back" else "not_applied"
    if soul == record["soul_before"] and source_after and key != "-":
        return "source_saved_host_pending"
    raise Conflict("unrecognized_target_content")
