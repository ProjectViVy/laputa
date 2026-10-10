# Stop after unresolved work effects

Reconcile retained an unknown/rejected WorkPatch receipt but still invoked reflection and applied later memory effects. It now stops that stage chain immediately, retaining the receipt and classifying the terminal owner outcome as recovery_required or partial. Only applied/no-change/submitted work can continue.

ImplementationRevision advances to diva-cognitive/v1-review-3. Old admitted operation identities are preserved, not rebound silently. The change is confined to the owning strategy and existing Domain/Model contract; no authority, persistence or scheduler was added.
