"""Explicitly enabled native SDK entry. Importing this module is passive."""

import copy
import json
import logging
from dataclasses import replace
from functools import partial
from pathlib import Path
from types import SimpleNamespace

from .apply import apply, resume_host
from .domain import BoundaryError, Cancelled, Conflict, digest
from .host import supports_persona_stage
from .markdown import bind_scope, read_sources, safe_read
from .proposal import propose
from .recovery import read_receipt

logger = logging.getLogger(__name__)


class PersonaController:
    """One fixed approved binding per registration; no persistent evidence store."""

    def __init__(self, ctx, config):
        self.ctx = ctx
        self.config = copy.deepcopy(config)
        if any(
            config.get(k) is not True
            for k in ("source_access_approved", "writes_approved")
        ):
            raise BoundaryError("explicit_source_and_write_consent_required")
        if type(config.get("model_use_approved")) is not bool:
            raise BoundaryError("explicit_model_consent_choice_required")
        if config["model_use_approved"] and any(
            not isinstance(config.get(k), str) or not config[k].strip()
            for k in ("provider", "model")
        ):
            raise BoundaryError("explicit_approved_provider_and_model_required")
        values = config.get("scope")
        if not isinstance(values, dict) or "binding_probe" in values:
            raise BoundaryError("explicit_scope_required")
        for key in ("home", "shared_root"):
            if (
                not isinstance(values.get(key), str)
                or not Path(values[key]).is_absolute()
            ):
                raise BoundaryError("absolute_scope_paths_required")
        self.scope = bind_scope(**values)
        self.scope = replace(
            self.scope,
            binding_probe=lambda: self.scope.binding_key if self.current() else "",
        )
        from agent.prompt_builder import _get_context_file_max_chars

        if self.scope.max_soul_chars > _get_context_file_max_chars():
            raise BoundaryError("native_identity_budget_exceeded")
        self.last_evidence = None
        self.latest_report = {}

    def current(self):
        from agent.prompt_builder import _get_context_file_max_chars

        return (
            self.ctx.get_config("persona", {}) == self.config
            and self.scope.max_soul_chars <= _get_context_file_max_chars()
        )

    def authorize(self, metadata):
        return (
            self.current()
            and metadata.get("home") == str(self.scope.home)
            and metadata.get("surface") == self.scope.surface
        )

    def report(self, state, *, tokens=0, classification=""):
        self.latest_report = {
            "state": state,
            "scope_digest": self.scope.binding_key,
            "total_tokens": tokens,
            "classification": classification,
        }
        logger.info(
            "Garden persona state=%s scope=%s tokens=%s",
            state,
            self.scope.binding_key,
            tokens,
        )
        return state

    def _adopt_recorded_target(self):
        record = read_receipt(self.scope)
        if record is None:
            return None
        from .recovery import validate_materials

        validate_materials(self.scope, record)
        if record["state"] in ("verified", "rolled_back"):
            expected = (
                record["soul_after"]
                if record["state"] == "verified"
                else record["soul_before"]
            )
            _, current = safe_read(self.scope.soul_path, 4 * self.scope.max_soul_chars)
            if current != expected:
                raise Conflict("recorded_soul_has_drifted")
            self.scope = replace(self.scope, managed_soul_digest=expected)
            return None
        if record["state"] == "source_pending":
            return record
        raise Conflict("inspect_unresolved_last_apply_before_new_attempt")

    def run(self, stage):
        tokens = 0
        try:
            if stage.cancelled():
                return self.report("cancelled")
            if (
                not self.current()
                or Path(stage.home).resolve() != self.scope.home
                or stage.surface != self.scope.surface
            ):
                return self.report("rejected")
            self.scope.assert_current()
            if self._adopt_recorded_target() is not None:
                result = resume_host(self.scope, enter_commit=stage.try_enter_commit)
                if result.state == "persisted_verified":
                    self.scope = replace(
                        self.scope, managed_soul_digest=result.soul_digest
                    )
                return self.report(result.state)
            snapshot = read_sources(self.scope)
            evidence = [
                {"item_id": i.item_id, "role": i.role, "text": i.text}
                for i in stage.items
            ]
            fingerprint = digest(
                json.dumps(
                    [(i["role"], i["text"]) for i in evidence], ensure_ascii=False
                ).encode()
            )
            candidate = None
            classification = "no_change"
            if (
                self.config["model_use_approved"]
                and evidence
                and fingerprint != self.last_evidence
            ):
                if stage.cancelled():
                    return self.report("cancelled")
                self.last_evidence = fingerprint
                # Explicit route overrides still pass through the SDK's native trust gates.
                llm = SimpleNamespace(
                    complete_structured=partial(
                        self.ctx.llm.complete_structured,
                        provider=self.config["provider"],
                        model=self.config["model"],
                        purpose="garden-persona",
                    )
                )
                outcome = propose(
                    self.scope, snapshot, evidence, llm, cancelled=stage.cancelled
                )
                tokens = outcome.total_tokens
                classification = outcome.state
                if outcome.state not in (
                    "proposal",
                    "no_change",
                    "route_user_fact",
                    "route_memory",
                    "route_skill",
                ):
                    return self.report(outcome.state, tokens=tokens)
                candidate = outcome.replacement
            if stage.cancelled():
                return self.report("cancelled", tokens=tokens)
            result = apply(
                self.scope, snapshot, candidate, enter_commit=stage.try_enter_commit
            )
            if result.state == "persisted_verified":
                self.scope = replace(self.scope, managed_soul_digest=result.soul_digest)
            return self.report(
                result.state, tokens=tokens, classification=classification
            )
        except Cancelled:
            return self.report("cancelled", tokens=tokens)
        except BoundaryError:
            return self.report("rejected", tokens=tokens)
        except ValueError:
            return self.report("rejected", tokens=tokens)
        except Exception:  # noqa: BLE001 - privacy boundary: never export provider exception text
            # Never copy exception/provider text, prompts or private paths into receipts/logs.
            return self.report("analysis_failed", tokens=tokens)


def register(ctx):
    if not supports_persona_stage(ctx):
        return None
    config = ctx.get_config("persona", {})
    if not isinstance(config, dict) or config.get("enabled") is not True:
        return None
    controller = PersonaController(ctx, config)
    return ctx.register_background_review_persona_stage(
        controller.run, authorize=controller.authorize, max_evidence_chars=16000
    )
