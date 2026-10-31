from dataclasses import replace

import pytest

from garden_hermes.domain import BoundaryError, digest
from garden_hermes.markdown import read_sources


@pytest.mark.parametrize("body", [b"", b" \n\t"])
def test_unassigned_mission_is_never_auto_generated_or_assembled(persona, body):
    path = persona.role / "MISSION.MD"
    path.write_bytes(body)
    scope = replace(persona.scope, approved_mission_digest=digest(body))
    with pytest.raises(BoundaryError):
        read_sources(scope)
    assert path.read_bytes() == body
    assert scope.soul_path.read_bytes() == b"Explicitly adopted synthetic target."
