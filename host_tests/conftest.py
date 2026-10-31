"""All data are synthetic and all files live under pytest's explicit scratch root."""

from types import SimpleNamespace

import pytest

from garden_hermes.domain import digest
from garden_hermes.markdown import bind_scope


@pytest.fixture
def persona(tmp_path):
    shared, home = tmp_path / "shared", tmp_path / "profile"
    role = shared / "personas" / "songbai"
    role.mkdir(parents=True)
    home.mkdir()
    bodies = {
        "MISSION.MD": b"Assigned synthetic mission.\r\n",
        "IDENTITY.MD": b"Plain identity.\n# Human headings are fine\n",
        "RELATIONSHIP.MD": b"An evolving relationship.",
        "REDLINE.MD": b"No privilege or privacy expansion.",
    }
    for name, value in bodies.items():
        (role / name).write_bytes(value)
    (home / "SOUL.md").write_bytes(b"Explicitly adopted synthetic target.")
    settings = {
        "shared_root": shared,
        "home": home,
        "persona_id": "songbai",
        "binding_revision": "revision-1",
        "consent_ref": "synthetic-consent",
        "surface": "cli",
        "approved_mission_digest": digest(bodies["MISSION.MD"]),
        "managed_soul_digest": digest((home / "SOUL.md").read_bytes()),
        "max_source_bytes": 20000,
        "max_soul_chars": 10000,
    }
    return SimpleNamespace(
        shared=shared,
        home=home,
        role=role,
        bodies=bodies,
        settings=settings,
        scope=bind_scope(**settings),
    )
