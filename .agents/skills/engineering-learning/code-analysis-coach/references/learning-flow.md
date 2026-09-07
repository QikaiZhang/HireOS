# Guided Learning Flow

## Use Only In A Practice Session

Use this flow only when the user explicitly asks to be taught, practise, or be checked. Ordinary code-reading questions should end once the user's question is answered.

1. Agree on one concrete target: a constructor, a request path, a message consumer, or a change point.
2. Give a short orientation: where it sits and what the user should notice first.
3. Trace one source-confirmed relationship at a time. Explain the local responsibility before introducing the next symbol.
4. Ask one small question that requires connecting the relationship, not recalling a name.
5. Use the answer to decide whether to restate with a smaller example, continue one hop, or discuss the design reason.
6. End with a plain-language recap of the reusable idea and one optional next path.

## Questions

Prefer this progression:

1. **Basic understanding:** Ask the user to reconstruct a concrete input-to-output path or identify an ownership boundary.
2. **Design reasoning:** Ask why a layer, transaction, cache, retry, lock, or state transition exists and what breaks without it.
3. **Scenario transfer:** Change a real condition, such as duplicate delivery, slow storage, downstream failure, or increased load, and ask how the design should respond.

Do not ask a trivia question such as "what does this method do?" when the answer is visible from its name. Do not ask more than one question at a time. Let the user choose whether to continue after the core explanation.

## Mastery Assessment

Assess demonstrated ability, not confidence language.

| Level | Evidence | Next coaching move |
| --- | --- | --- |
| `CONCEPT_UNDERSTOOD` | Can explain a local concept in the current context. | Connect it to the actual call path. |
| `CODE_UNDERSTOOD` | Can reconstruct key control/data flow and ownership. | Ask why the boundary or mechanism exists. |
| `ARCHITECTURE_UNDERSTOOD` | Can relate the module to callers, dependencies, and failure boundaries. | Explore alternatives and tradeoffs. |
| `DESIGN_REASONING` | Can defend the selected design and name its costs. | Introduce a changed scenario. |
| `INTERVIEW_READY` | Can transfer the reasoning to a new scenario and give a concise, evidence-based project narrative. | Record a compact interview artifact if it has durable value. |

Use these levels only when the user asks for an assessment. Otherwise use the user's answer to choose the next explanation; do not announce a grade or a gap list. Do not treat unfamiliar terminology as lack of understanding when the source has not yet established it.
