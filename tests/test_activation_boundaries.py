import pytest

from garden_hermes.activation import observe_identity


@pytest.mark.parametrize("overlay", [None, True])
def test_unknown_or_active_overlay_is_not_attested(persona, overlay):
    result = observe_identity(
        persona.scope,
        b"Full identity",
        [{"role": "system", "content": "Full identity"}],
        request_home=persona.home,
        request_session_id="new",
        previous_session_id="old",
        overlay_active=overlay,
    )
    assert result["state"] == "activation_unverified"


@pytest.mark.parametrize(
    "messages",
    [
        [{"role": "system", "content": "Full"}],
        [{"role": "system", "content": "Full identity Full identity"}],
        [{"role": "system", "content": "Overlay before Full identity"}],
        [{"role": "developer", "content": "Full identity"}],
    ],
)
def test_incomplete_duplicate_or_unsupported_identity_is_unverified(persona, messages):
    result = observe_identity(
        persona.scope,
        b"Full identity",
        messages,
        request_home=persona.home,
        request_session_id="new",
        previous_session_id="old",
        overlay_active=False,
    )
    assert result["state"] == "activation_unverified"
