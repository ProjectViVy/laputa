"""Private, opt-in observation only: never installs hooks or refreshes sessions."""

from pathlib import Path

from .domain import digest


def observe_identity(
    scope,
    expected,
    request_messages,
    *,
    request_home,
    request_session_id,
    previous_session_id,
    overlay_active=None,
):
    """Call with transport-owned request data, not a model's claim of activation.

    Only hash/status metadata leaves this pure helper. Supports normalized chat
    system messages; an unsupported/ambiguous transport is unverified, never effective.
    """
    result = {
        "state": "activation_unverified",
        "scope_digest": scope.binding_key,
        "identity_digest": None,
    }
    if (
        overlay_active is not False
        or not request_session_id
        or not previous_session_id
        or Path(request_home).resolve() != scope.home
    ):
        return result
    text = expected.decode("utf-8").replace("\r\n", "\n").replace("\r", "\n").strip()
    if not text:
        return result
    systems = [
        m.get("content")
        for m in request_messages
        if isinstance(m, dict) and m.get("role") == "system"
    ]
    if len(systems) != 1 or not isinstance(systems[0], str):
        return result
    body = systems[0]
    if not body.startswith(text) or body.count(text) != 1:
        return result
    result.update(
        state="observed_new_session"
        if request_session_id != previous_session_id
        else "observed_same_session",
        identity_digest=digest(body[: len(text)].encode("utf-8")),
    )
    return result
