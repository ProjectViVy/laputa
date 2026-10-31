from types import SimpleNamespace

import pytest


@pytest.mark.parametrize("supported", [False, True])
def test_disabled_or_unsupported_entry_is_passive(supported):
    from garden_hermes.plugin import register

    def forbidden(*args, **kwargs):
        raise AssertionError("No enabled stage or profile read is permitted")

    ctx = SimpleNamespace(
        background_review_persona_stage_version=1 if supported else 0,
        register_background_review_persona_stage=forbidden,
        get_config=(lambda *a: {}) if supported else forbidden,
    )
    assert register(ctx) is None
