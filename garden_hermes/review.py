"""Business validation is mandatory even without the host's optional jsonschema."""

import re

from .domain import AUTO_FILES, BoundaryError, FileReplacement


class NeedsUserAuthorization(BoundaryError):
    pass


def validate_replacement(scope, snapshot, value, evidence):
    allowed = {
        "result",
        "file_key",
        "replacement_text",
        "source_item_ids",
        "rationale",
        "counterevidence",
    }
    if (
        not isinstance(value, dict)
        or set(value) - allowed
        or value.get("result") != "proposal"
    ):
        raise BoundaryError("invalid_output_fields")
    if snapshot.binding_key != scope.binding_key:
        raise BoundaryError("wrong_binding")
    key = value.get("file_key")
    if key in ("MISSION.MD", "REDLINE.MD"):
        raise NeedsUserAuthorization("protected_file")
    if not isinstance(key, str) or key not in AUTO_FILES:
        raise BoundaryError("file_not_auto_writable")
    body = value.get("replacement_text")
    if not isinstance(body, str) or not body.strip():
        raise BoundaryError("empty_replacement")
    # Conservative escalation heuristic, not a semantic sandbox or a body schema.
    if re.search(
        r"\b(?:permissions?|admin(?:istrator)?s?|privileges?|credentials?|mission|privacy|system prompt|api[ _-]?keys?)\b|权限|授权|管理员|凭据|隐私|使命|系统提示",
        body,
        re.IGNORECASE,
    ):
        raise NeedsUserAuthorization("authority_sensitive_content")
    try:
        body = body.encode("utf-8")
    except UnicodeEncodeError:
        raise BoundaryError("invalid_utf8") from None
    if len(body) > scope.max_source_bytes:
        raise BoundaryError("source_byte_budget_exceeded")
    allowed_ids = {item["item_id"] for item in evidence}
    for name in ("source_item_ids", "counterevidence"):
        ids = value.get(name, [])
        if (
            not isinstance(ids, list)
            or any(not isinstance(x, str) or x not in allowed_ids for x in ids)
            or len(set(ids)) != len(ids)
            or (name == "source_item_ids" and not ids)
        ):
            raise BoundaryError("unattributable_candidate")
    rationale = value.get("rationale", "")
    if not isinstance(rationale, str) or len(rationale) > 2000:
        raise BoundaryError("invalid_rationale")
    return FileReplacement(
        scope.binding_key,
        key,
        snapshot.source_versions[key],
        body,
        tuple(value["source_item_ids"]),
        rationale,
        tuple(value.get("counterevidence", [])),
    )
