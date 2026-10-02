package diva

// The prompts and output schemas are part of the strategy's identity:
// changing them changes the strategy digest. They live in the library so
// hosts never ship parallel copies.

// PromptReconcile asks for a bounded Work patch over the scoped ACTMEM
// snapshot plus the evidence batch. The model never emits scope, identity
// or destination — the binding is added by the stage.
const PromptReconcile = `You are the work-reconciliation stage of the diva/v1 evolution strategy.

Input JSON: {"input": {"binding": ..., "window": ...}, "batch": {"entries": [...], "persona": [...], "sources": [...]}}.

Produce a WorkPatch whose changes bring the scoped Work section in line with
the committed activity. Every change's sources must cite batch source ids.
If nothing actionable changed, return {"base_revision": <activity_revision>,
"changes": []}.

Never propose Mission or Dream writes. Never write Pulse or Recap entries.`

// PromptReflect asks for typed candidates only.
const PromptReflect = `You are the reflection stage of the diva/v1 evolution strategy.

Emit a JSON object with exactly one of:
- "candidates": an ordered list of typed candidates, each exactly one of
  {"kind":"memory_mutation","memory_mutation":{...}},
  {"kind":"persona_request","persona_request":{...}},
  {"kind":"capability_proposal","capability_proposal":{...}},
  {"kind":"reflection_note","reflection_note":{...}}
- "no_change_reason": a short string when there is nothing worth doing.

Rules: cite batch source ids; memory_mutation requires record_id and an
inference class; persona_request kind is one of identity|relationship|dark|
user_observations|world. Mission and Dream writes do not exist.`

// SchemaWorkPatch is the reconcile output schema.
const SchemaWorkPatch = `{"type":"object","required":["base_revision","changes"],"properties":{"base_revision":{"type":"integer","minimum":0},"changes":{"type":"array","items":{"type":"object","required":["kind","field"],"properties":{"kind":{"enum":["add","replace","complete","drop"]},"entry_id":{"type":"string"},"field":{"enum":["goal","open","next","constraints","pointers"]},"body":{"type":"string"},"sources":{"type":"array"}}}}},"additionalProperties":false}`

// SchemaReflectionOutput is the reflect output schema.
const SchemaReflectionOutput = `{"type":"object","properties":{"candidates":{"type":"array"},"no_change_reason":{"type":"string"}},"additionalProperties":false}`

// ImplementationRevision identifies this strategy build; it participates
// in the strategy digest alongside the definition and prompts.
const ImplementationRevision = "diva-cognitive/v1-review-1"

// PromptBundle returns the canonical identity bytes the strategy digest
// covers: every prompt and schema in declaration order.
func PromptBundle() []byte {
	return []byte(PromptReconcile + "\x00" + PromptReflect + "\x00" + SchemaWorkPatch + "\x00" + SchemaReflectionOutput + "\x00" + ImplementationRevision)
}
