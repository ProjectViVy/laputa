import os

import pytest

from garden_hermes.apply import apply
from garden_hermes.assemble import assemble
from garden_hermes.domain import BoundaryError
from garden_hermes.markdown import read_sources


def test_alias_planted_after_publication_is_not_verified(
    persona, tmp_path, monkeypatch
):
    snapshot = read_sources(persona.scope)
    expected = assemble(persona.scope, snapshot, None)
    foreign = tmp_path / "foreign-same-bytes.md"
    foreign.write_bytes(expected)
    original = os.replace

    def publish(source, target):
        original(source, target)
        if str(target) == str(persona.scope.soul_path):
            persona.scope.soul_path.unlink()
            persona.scope.soul_path.symlink_to(foreign)

    monkeypatch.setattr(os, "replace", publish)
    with pytest.raises(BoundaryError):
        apply(persona.scope, snapshot, None, enter_commit=lambda: True)
    assert foreign.read_bytes() == expected
