"""Host-independent contracts. Source bodies are opaque bytes, not schemas."""

import hashlib
from collections.abc import Callable, Mapping
from dataclasses import dataclass, field
from pathlib import Path
from types import MappingProxyType

ASSEMBLY_FILES = (
    "MISSION.MD",
    "IDENTITY.MD",
    "RELATIONSHIP.MD",
    "REDLINE.MD",
    "DREAM.MD",
    "DARK.MD",
)
AUTO_FILES = frozenset(("IDENTITY.MD", "RELATIONSHIP.MD", "DREAM.MD", "DARK.MD"))
OPTIONAL_FILES = frozenset(("DREAM.MD", "DARK.MD"))


class BoundaryError(ValueError):
    """A scope, permission, version or file boundary failed closed."""


class Conflict(BoundaryError):
    pass


class Cancelled(BoundaryError):
    pass


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


@dataclass(frozen=True)
class BoundScope:
    shared_root: Path
    source_dir: Path
    home: Path
    persona_id: str
    binding_revision: str
    consent_ref: str
    surface: str
    approved_mission_digest: str
    managed_soul_digest: str
    max_source_bytes: int
    max_soul_chars: int
    include_host: bool = False
    binding_probe: Callable[[], str] | None = field(
        default=None, repr=False, compare=False
    )

    @property
    def soul_path(self) -> Path:
        return self.home / "SOUL.md"

    @property
    def recovery_dir(self) -> Path:
        return self.home / "plugin-data" / "garden" / "last-apply" / self.persona_id

    @property
    def binding_key(self) -> str:
        values = (
            str(self.shared_root),
            str(self.source_dir),
            str(self.home),
            self.persona_id,
            self.binding_revision,
            self.consent_ref,
            self.surface,
            self.approved_mission_digest,
            str(self.max_source_bytes),
            str(self.max_soul_chars),
            str(self.include_host),
        )
        return digest("\0".join(values).encode())

    def assert_current(self) -> None:
        if self.binding_probe is not None and self.binding_probe() != self.binding_key:
            raise Conflict("stale_role_binding")


@dataclass(frozen=True)
class SourceSnapshot:
    binding_key: str
    file_bytes: Mapping[str, bytes]
    source_versions: Mapping[str, str | None]
    host_bytes: bytes
    host_version: str | None
    soul_before: bytes

    def __post_init__(self):
        object.__setattr__(self, "file_bytes", MappingProxyType(dict(self.file_bytes)))
        object.__setattr__(
            self, "source_versions", MappingProxyType(dict(self.source_versions))
        )


@dataclass(frozen=True)
class FileReplacement:
    binding_key: str
    file_key: str
    expected_file_version: str | None
    replacement: bytes
    source_item_ids: tuple[str, ...] = ()
    rationale: str = ""
    counterevidence: tuple[str, ...] = ()


@dataclass(frozen=True)
class ApplyResult:
    operation_id: str
    state: str
    source_digest: str | None = None
    soul_digest: str | None = None
    reason: str = ""
