"""Hermetic, serial verification of synthetic persona development; no installation."""

import argparse
import json
import os
import shutil
import subprocess
import sys
import tomllib
import xml.etree.ElementTree as ET
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--host-source", type=Path, required=True)
    parser.add_argument("--host-python", type=Path, required=True)
    parser.add_argument("--component-python", type=Path, default=Path(sys.executable))
    parser.add_argument("--scratch", type=Path, required=True)
    parser.add_argument(
        "--installed",
        action="store_true",
        help="Require imports from the selected environments, not this source tree",
    )
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    source = args.host_source.resolve()
    scratch = args.scratch.resolve()
    for protected in (root, source, Path.home()):
        if scratch == protected or scratch in protected.parents:
            parser.error("scratch must not contain a source tree or home")
    scratch.mkdir(parents=True, exist_ok=True, mode=0o700)
    assets = scratch / "test-assets"
    for name in ("tests", "host_tests", "tools"):
        shutil.copytree(
            root / name,
            assets / name,
            dirs_exist_ok=True,
            ignore=shutil.ignore_patterns("__pycache__", ".pytest_cache"),
        )
    with (source / "pyproject.toml").open("rb") as stream:
        markers = (
            tomllib.load(stream)
            .get("tool", {})
            .get("pytest", {})
            .get("ini_options", {})
            .get("markers", [])
        )
    config = assets / "pytest.ini"
    config.write_text(
        "[pytest]\nmarkers =\n" + "".join("    " + line + "\n" for line in markers),
        encoding="utf-8",
    )
    env = {
        "PATH": os.environ.get("PATH", os.defpath),
        "HOME": str(scratch / "home"),
        "HERMES_HOME": str(scratch / "profile"),
        "TMPDIR": str(scratch / "tmp"),
        "TMP": str(scratch / "tmp"),
        "TZ": "UTC",
        "LANG": "C.UTF-8",
        "PYTHONHASHSEED": "0",
        "PYTHONNOUSERSITE": "1",
    }
    for key in ("HOME", "HERMES_HOME", "TMPDIR"):
        Path(env[key]).mkdir(parents=True, exist_ok=True, mode=0o700)
    host_files = sorted((source / "tests/agent").glob("*background*review*.py"))
    host_files += sorted((source / "tests/agent").glob("test_turn_finalizer*.py"))
    host_files += [
        source / relative
        for relative in (
            "tests/agent/test_persona_stage.py",
            "tests/agent/test_plugin_llm.py",
            "tests/agent/test_plugin_llm_task_routing.py",
            "tests/agent/test_profile_home_override_precedence.py",
            "tests/plugins/test_plugin_loader.py",
            "tests/plugins/test_plugin_loader_unreadable.py",
            "tests/plugins/test_plugin_paths_follow_profile.py",
            "tests/plugins/test_multiplex_plugin_cache_scope.py",
        )
    ]
    host_files += sorted((assets / "host_tests").glob("test_*.py"))
    suites = (
        (
            "component",
            args.component_python,
            [assets / "tests"],
            scratch / "component-cwd",
        ),
        ("host", args.host_python, sorted(set(host_files)), source),
    )
    summary = {}
    for name, interpreter, files, cwd in suites:
        cwd.mkdir(parents=True, exist_ok=True)
        call_env = dict(env)
        paths = ([str(source)] if name == "host" else []) + (
            [] if args.installed else [str(root)]
        )
        if paths:
            call_env["PYTHONPATH"] = os.pathsep.join(paths)
        if args.installed:
            probe = "import pathlib,sys,garden_hermes; assert pathlib.Path(garden_hermes.__file__).resolve().is_relative_to(pathlib.Path(sys.prefix).resolve()), garden_hermes.__file__"
            subprocess.run(
                [str(interpreter), "-c", probe],
                cwd=cwd,
                env=call_env,
                check=True,
                timeout=30,
            )
        junit = scratch / (name + ".xml")
        command = [
            str(interpreter),
            "-m",
            "pytest",
            "-q",
            "--tb=short",
            "--import-mode=importlib",
            "-c",
            str(config),
            "--basetemp=" + str(scratch / (name + "-tmp")),
            "--junitxml=" + str(junit),
            *map(str, files),
        ]
        result = subprocess.run(
            command, cwd=cwd, env=call_env, check=False, timeout=300
        )
        if junit.exists():
            cases = list(ET.parse(junit).iter("testcase"))
            summary[name] = {
                "cases": len(cases),
                "failed": sum(
                    c.find("failure") is not None or c.find("error") is not None
                    for c in cases
                ),
                "skipped": sum(c.find("skipped") is not None for c in cases),
                "exit_code": result.returncode,
            }
        else:
            summary[name] = {"exit_code": result.returncode}
        if result.returncode:
            print(json.dumps(summary, indent=2))
            return result.returncode
    (scratch / "summary.json").write_text(
        json.dumps(summary, indent=2) + "\n", encoding="utf-8"
    )
    print(json.dumps(summary, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
