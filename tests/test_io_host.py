import importlib
import os


def test_atomic_plaintext_replace_keeps_exact_body_and_private_mode(
    tmp_path, monkeypatch
):
    m = importlib.import_module("garden_hermes.markdown")
    target = tmp_path / "IDENTITY.MD"
    target.write_bytes(b"old")
    body = b"No special headings.\r\n---\nany: text\n"
    calls = []
    original = os.replace

    def counted(a, b):
        calls.append((a, b))
        return original(a, b)

    monkeypatch.setattr(os, "replace", counted)
    m.atomic_replace(target, body)
    assert target.read_bytes() == body
    assert target.stat().st_mode & 0o777 == 0o600
    assert len(calls) == 1
    assert list(tmp_path.iterdir()) == [target]


def test_capability_probe_is_passive_and_versioned():
    h = importlib.import_module("garden_hermes.host")

    class Legacy:
        pass

    class Supported:
        background_review_persona_stage_version = 1

        def register_background_review_persona_stage(self, *args, **kwargs):
            raise AssertionError(
                "Capability inspection must not register or run anything"
            )

    assert not h.supports_persona_stage(Legacy())
    assert h.supports_persona_stage(Supported())
