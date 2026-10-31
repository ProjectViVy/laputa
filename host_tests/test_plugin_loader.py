"""Exercise the real manifest parser, module loader and owner-ledger unload."""

import shutil
from pathlib import Path


def test_packaged_bundle_loads_disabled_without_source_access(tmp_path, monkeypatch):
    from hermes_cli.plugins import PluginManager
    from hermes_cli.plugins_loader import _plugin_home_scope
    from hermes_cli.plugins_manifest import parse_manifest_file

    import garden_hermes

    home = tmp_path / "home"
    home.mkdir()
    monkeypatch.setenv("HERMES_HOME", str(home))
    source = Path(garden_hermes.__file__).parent / "hermes_plugin"
    target = home / "plugins" / "garden-persona"
    shutil.copytree(source, target)
    manifest = parse_manifest_file(target / "plugin.yaml", target, "user", "")
    assert manifest is not None and manifest.name == "garden-persona"
    with _plugin_home_scope(home):
        manager = PluginManager(scope_key=str(home))
        manager._load_plugin(manifest)
        assert manager._plugins[manifest.key].enabled
        assert manager.unload(manifest.key)
    assert not (home / "plugin-data" / "garden").exists()


def test_enabled_loader_unload_revokes_captured_stage(persona, monkeypatch):
    import json

    from agent import persona_stage
    from agent.background_review import _BackgroundReviewRun
    from hermes_cli.plugins import PluginManager
    from hermes_cli.plugins_loader import _plugin_home_scope
    from hermes_cli.plugins_manifest import parse_manifest_file

    import garden_hermes

    settings = {
        k: str(v) if isinstance(v, Path) else v for k, v in persona.settings.items()
    }
    config = {
        "enabled": True,
        "source_access_approved": True,
        "writes_approved": True,
        "model_use_approved": False,
        "scope": settings,
    }
    (persona.home / "config.yaml").write_text(
        json.dumps(
            {
                "plugins": {
                    "entries": {
                        "garden-persona": {
                            "enabled": True,
                            "settings": {"persona": config},
                        }
                    }
                }
            }
        ),
        encoding="utf-8",
    )
    monkeypatch.setenv("HERMES_HOME", str(persona.home))
    target = persona.home / "plugins" / "garden-persona"
    shutil.copytree(Path(garden_hermes.__file__).parent / "hermes_plugin", target)
    manifest = parse_manifest_file(target / "plugin.yaml", target, "user", "")
    with _plugin_home_scope(persona.home):
        manager = PluginManager(scope_key=str(persona.home))
        manager._load_plugin(manifest)
        assert manager._plugins[manifest.key].enabled
        request = persona_stage.capture_turn(
            home=persona.home,
            surface="cli",
            session_id="s",
            turn_id="t",
            user_text="Synthetic visible user",
            assistant_text="Synthetic visible answer",
        )
        assert request is not None
        assert manager.unload(manifest.key)
        assert (
            persona_stage.invoke(request, _BackgroundReviewRun(), {"completed": True})
            == "cancelled"
        )
    assert not persona.scope.recovery_dir.exists()
