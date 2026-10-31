import json
from pathlib import Path
from types import SimpleNamespace

from hermes_cli.plugins_loader import _plugin_home_scope

from garden_hermes.plugin import PersonaController


def test_native_budget_is_rechecked_after_registration(persona):
    values = {
        k: str(v) if isinstance(v, Path) else v for k, v in persona.settings.items()
    }
    config = {
        "enabled": True,
        "source_access_approved": True,
        "writes_approved": True,
        "model_use_approved": False,
        "scope": values,
    }
    sdk = SimpleNamespace(get_config=lambda *args: config)
    with _plugin_home_scope(persona.home):
        controller = PersonaController(sdk, config)
        (persona.home / "config.yaml").write_text(
            json.dumps({"context_file_max_chars": 30}), encoding="utf-8"
        )
        stage = SimpleNamespace(
            home=persona.home,
            surface="cli",
            items=[],
            cancelled=lambda: False,
            try_enter_commit=lambda: True,
        )
        assert controller.run(stage) == "rejected"
    assert (
        persona.scope.soul_path.read_bytes() == b"Explicitly adopted synthetic target."
    )
    assert not persona.scope.recovery_dir.exists()
