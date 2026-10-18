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
  {"kind":"skill_proposal","skill_proposal":{...}},
  {"kind":"reflection_note","reflection_note":{...}}
- "no_change_reason": a short string when there is nothing worth doing.

Rules: cite only batch source ids. For a skill_proposal, update the selected
local Skill when it covers the same durable capability; create one only when
none of the listed Skills fits. An update must use the selected Skill's exact
name and hash and incorporate its full content. Include proposed_content, a
concise rationale and source references. A proposal is only a candidate; the
host requires explicit human approval before applying it. Never claim it was
installed. capability_proposal remains EvoMap-only. memory_mutation requires
record_id and an inference class; persona_request kind is one of
identity|relationship|dark|user_observations|world. Mission and Dream writes do not exist.`

// PromptSkillSelect asks for one exact existing Skill name or an empty name.
const PromptSkillSelect = `You are selecting a local Skill that could be improved by the current evidence.

Choose one listed Skill only when it covers the same durable capability as the
work in the evidence. Prefer a relevant existing Skill so later learning adds
a revision to the same identity. Return an empty skill_name when none fits;
the reflection stage may then propose a new Skill if the task class is novel.
Never select a Skill by guessing its name.`

// SchemaWorkPatch is the reconcile output schema.
const SchemaWorkPatch = `{"type":"object","required":["base_revision","changes"],"properties":{"base_revision":{"type":"integer","minimum":0},"changes":{"type":"array","items":{"type":"object","required":["kind","field"],"properties":{"kind":{"enum":["add","replace","complete","drop"]},"entry_id":{"type":"string"},"field":{"enum":["goal","open","next","constraints","pointers"]},"body":{"type":"string"},"sources":{"type":"array"}}}}},"additionalProperties":false}`

// SchemaReflectionOutput is the reflect output schema.
const SchemaReflectionOutput = `{"type":"object","properties":{"candidates":{"type":"array"},"no_change_reason":{"type":"string"}},"additionalProperties":false}`

// SchemaSkillSelect is the closed exact-name selection result.
const SchemaSkillSelect = `{"type":"object","required":["skill_name"],"properties":{"skill_name":{"type":"string"}},"additionalProperties":false}`

// ImplementationRevision identifies this strategy build; it participates
// in the strategy digest alongside the definition and prompts.
const ImplementationRevision = "diva-cognitive/v1-review-4"

// PromptBundle returns the canonical identity bytes the strategy digest
// covers: every prompt and schema in declaration order.
func PromptBundle() []byte {
	return []byte(PromptReconcile + "\x00" + PromptSkillSelect + "\x00" + PromptReflect + "\x00" + SchemaWorkPatch + "\x00" + SchemaSkillSelect + "\x00" + SchemaReflectionOutput + "\x00" + ImplementationRevision)
}
