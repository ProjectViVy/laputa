import pytest

from garden_hermes.domain import BoundaryError
from garden_hermes.markdown import read_sources


def test_world_writable_source_file_is_not_trusted(persona):
    (persona.role / "IDENTITY.MD").chmod(0o666)
    with pytest.raises(BoundaryError):
        read_sources(persona.scope)
