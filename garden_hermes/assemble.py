"""Deterministic, whole-body assembly. No body parsing or native store mutation."""

from .domain import ASSEMBLY_FILES, AUTO_FILES, BoundaryError, Conflict, digest


def assemble(scope, snapshot, replacement=None):
    scope.assert_current()
    if snapshot.binding_key != scope.binding_key:
        raise Conflict("wrong_role_binding")
    if digest(snapshot.file_bytes["MISSION.MD"]) != scope.approved_mission_digest:
        raise Conflict("mission_version_not_authorized")
    bodies = dict(snapshot.file_bytes)
    if replacement is not None:
        if replacement.binding_key != scope.binding_key:
            raise Conflict("wrong_role_binding")
        if replacement.file_key not in AUTO_FILES:
            raise BoundaryError("file_not_auto_writable")
        if (
            replacement.expected_file_version
            != snapshot.source_versions[replacement.file_key]
        ):
            raise Conflict("stale_candidate")
        if (
            not isinstance(replacement.replacement, bytes)
            or not replacement.replacement.strip()
            or len(replacement.replacement) > scope.max_source_bytes
        ):
            raise BoundaryError("invalid_replacement")
        bodies[replacement.file_key] = replacement.replacement
    parts = [bodies[k] for k in ASSEMBLY_FILES if bodies[k]]
    if scope.include_host and snapshot.host_bytes:
        parts.append(snapshot.host_bytes)
    result = b"\n\n".join(parts)
    try:
        text = result.decode("utf-8")
    except UnicodeDecodeError:
        raise BoundaryError("invalid_utf8") from None
    if len(text) > scope.max_soul_chars:
        raise BoundaryError("soul_character_budget_exceeded")
    return result
