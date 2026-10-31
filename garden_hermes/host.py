"""Passive SDK capability detection; no configuration, profile reads or imports."""


def supports_persona_stage(ctx) -> bool:
    return (
        type(getattr(ctx, "background_review_persona_stage_version", None)) is int
        and ctx.background_review_persona_stage_version == 1
        and callable(getattr(ctx, "register_background_review_persona_stage", None))
    )
