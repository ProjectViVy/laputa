from garden_hermes.domain import digest


def test_activation_observation_requires_real_request_identity_and_new_boundary(
    persona,
):
    from garden_hermes.activation import observe_identity

    expected = b"New synthetic identity.\r\nMission unchanged."
    request = [
        {
            "role": "system",
            "content": "New synthetic identity.\nMission unchanged.\n\nNative guidance.",
        }
    ]
    result = observe_identity(
        persona.scope,
        expected,
        request,
        request_home=persona.home,
        request_session_id="new",
        previous_session_id="old",
        overlay_active=False,
    )
    assert result["state"] == "observed_new_session"
    assert result["identity_digest"] == digest(
        expected.decode().replace("\r\n", "\n").strip().encode()
    )
    stale = observe_identity(
        persona.scope,
        expected,
        [{"role": "system", "content": "Old cached identity."}],
        request_home=persona.home,
        request_session_id="old",
        previous_session_id="old",
        overlay_active=False,
    )
    assert stale["state"] == "activation_unverified"
    foreign = observe_identity(
        persona.scope,
        expected,
        request,
        request_home=persona.shared,
        request_session_id="new",
        previous_session_id="old",
        overlay_active=False,
    )
    assert foreign["state"] == "activation_unverified"
    assert "New synthetic identity" not in str(result)
