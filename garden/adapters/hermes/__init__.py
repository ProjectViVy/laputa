def register(ctx):
    # Native standalone plugin activation is opt-in. Import/discovery performs
    # no registration, secret read, request, profile creation or file write.
    names = ("endpoint", "binding_id", "credential_file", "host_instance", "hermes_home")
    values = {name: ctx.get_config(name) for name in names}
    if any(not isinstance(value, str) or not value.strip() for value in values.values()):
        return
    from pathlib import Path
    from hermes_constants import get_hermes_home
    from hermes_cli.version_info import get_version_info
    from .garden_prompt import GardenPromptConfig, make_prompt_hook
    try:
        # Hermes managers scope registration and invocation to their own home.
        # Require the explicitly selected home; never manufacture another profile.
        home = Path(values["hermes_home"]).resolve(strict=True)
        if home != get_hermes_home().resolve(strict=True):
            return
        version = get_version_info().display_version
        if version != "0.21.7":
            return
        ctx.register_hook("pre_llm_call", make_prompt_hook(GardenPromptConfig(**values, host_version=version)))
    except (OSError, ValueError):
        return
