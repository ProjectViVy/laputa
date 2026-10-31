"""Explicit role/file reads only. Never parse headings or inline metadata."""

import os
import re
import stat
from pathlib import Path

from .domain import (
    ASSEMBLY_FILES,
    OPTIONAL_FILES,
    BoundaryError,
    BoundScope,
    Conflict,
    SourceSnapshot,
    digest,
)


def reject_symlinks(path: Path) -> None:
    for part in (path, *path.parents):
        if part.is_symlink():
            raise BoundaryError("symlink_not_supported")


def safe_read(
    path: Path, limit: int, *, optional: bool = False
) -> tuple[bytes, str | None]:
    reject_symlinks(path)
    try:
        fd = os.open(
            path,
            os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_NONBLOCK", 0),
        )
    except FileNotFoundError:
        if optional:
            return b"", None
        raise BoundaryError("required_file_missing") from None
    except OSError:
        raise BoundaryError("file_open_refused") from None
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if info.st_mode & stat.S_IWOTH:
            raise BoundaryError("world_writable_file_refused")
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
            raise BoundaryError("not_an_unaliased_regular_file")
        data = stream.read(limit + 1)
    if len(data) > limit:
        raise BoundaryError("source_byte_budget_exceeded")
    try:
        data.decode("utf-8")
    except UnicodeDecodeError:
        raise BoundaryError("invalid_utf8") from None
    return data, digest(data)


def bind_scope(
    *,
    shared_root,
    home,
    persona_id,
    binding_revision,
    consent_ref,
    surface,
    approved_mission_digest,
    managed_soul_digest,
    max_source_bytes,
    max_soul_chars,
    include_host=False,
    binding_probe=None,
) -> BoundScope:
    if (
        not isinstance(persona_id, str)
        or not persona_id
        or persona_id in (".", "..")
        or any(c in persona_id for c in ("/", "\\", "\0"))
    ):
        raise BoundaryError("invalid_persona_id")
    if (
        surface not in ("cli", "local")
        or any(
            not isinstance(v, str) or not v.strip()
            for v in (consent_ref, binding_revision)
        )
        or (binding_probe is not None and not callable(binding_probe))
    ):
        raise BoundaryError("untrusted_or_unconsented_surface")
    for value in (approved_mission_digest, managed_soul_digest):
        if not isinstance(value, str) or re.fullmatch(r"[0-9a-f]{64}", value) is None:
            raise BoundaryError("explicit_version_required")
    if any(type(n) is not int or n <= 0 for n in (max_source_bytes, max_soul_chars)):
        raise BoundaryError("explicit_positive_budget_required")
    if type(include_host) is not bool:
        raise BoundaryError("invalid_host_option")
    root, target = Path(shared_root).absolute(), Path(home).absolute()
    source = root / "personas" / persona_id
    for p in (root, root / "personas", target, source):
        reject_symlinks(p)
        if not p.is_dir():
            raise BoundaryError("explicit_directory_required")
        if p.stat().st_mode & stat.S_IWOTH:
            raise BoundaryError("world_writable_scope_refused")
    root, target, source = root.resolve(), target.resolve(), source.resolve()
    if (
        source == target
        or source.parent != root / "personas"
        or source.name != persona_id
    ):
        raise BoundaryError("role_directory_mismatch")
    scope = BoundScope(
        root,
        source,
        target,
        persona_id,
        binding_revision,
        consent_ref,
        surface,
        approved_mission_digest,
        managed_soul_digest,
        max_source_bytes,
        max_soul_chars,
        include_host,
        binding_probe,
    )
    scope.assert_current()
    return scope


def read_sources(scope: BoundScope) -> SourceSnapshot:
    scope.assert_current()
    bodies, versions = {}, {}
    for key in ASSEMBLY_FILES:
        bodies[key], versions[key] = safe_read(
            scope.source_dir / key,
            scope.max_source_bytes,
            optional=key in OPTIONAL_FILES,
        )
    if not bodies["MISSION.MD"].decode("utf-8").strip():
        raise BoundaryError("mission_assignment_required")
    if versions["MISSION.MD"] != scope.approved_mission_digest:
        raise Conflict("mission_version_not_authorized")
    if scope.include_host:
        host, host_version = safe_read(
            scope.home / "plugin-data" / "garden" / "HOST.md",
            scope.max_source_bytes,
            optional=True,
        )
    else:
        host, host_version = b"", None
    soul, soul_version = safe_read(scope.soul_path, 4 * scope.max_soul_chars)
    if len(soul.decode("utf-8")) > scope.max_soul_chars:
        raise BoundaryError("soul_character_budget_exceeded")
    if soul_version != scope.managed_soul_digest:
        raise Conflict("unmanaged_or_drifted_soul")
    scope.assert_current()
    return SourceSnapshot(scope.binding_key, bodies, versions, host, host_version, soul)


def atomic_replace(path: Path, data: bytes) -> None:
    """One replacement; a post-replace failure does not prove no write."""
    with prepared_replace(path, data) as publish:
        publish()


from contextlib import contextmanager


@contextmanager
def prepared_replace(path: Path, data: bytes):
    """Write/fsync complete bytes before yielding the short publication callback."""
    import tempfile

    path = Path(path)
    reject_symlinks(path)
    data.decode("utf-8")
    fd, name = tempfile.mkstemp(prefix=".garden-", suffix=".md", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as stream:
            os.fchmod(stream.fileno(), 0o600)
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())

        def publish():
            reject_symlinks(path)
            os.replace(name, path)
            directory = os.open(
                path.parent, os.O_RDONLY | getattr(os, "O_DIRECTORY", 0)
            )
            try:
                os.fsync(directory)
            finally:
                os.close(directory)

        yield publish
    finally:
        if os.path.exists(name):
            os.unlink(name)
