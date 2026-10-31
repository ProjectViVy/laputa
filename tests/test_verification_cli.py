"""The verification entry is a developer tool, not a runtime worker."""

import subprocess
import sys
from pathlib import Path


def test_verification_cli_help_is_passive():
    script = Path(__file__).resolve().parents[1] / "tools" / "verify.py"
    result = subprocess.run(
        [sys.executable, str(script), "--help"],
        capture_output=True,
        text=True,
        check=False,
    )
    assert result.returncode == 0
    assert "synthetic" in result.stdout
