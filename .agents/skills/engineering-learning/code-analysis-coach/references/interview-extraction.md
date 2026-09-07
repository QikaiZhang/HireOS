# Interview Extraction

## Separate The Two Views

Keep two distinct answers for the same module:

- **Development view:** Where to change, which inputs/outputs/contracts matter, the real path, operational constraints, and risks.
- **Interview view:** Problem, constraints, design choice, alternatives, tradeoffs, failure behavior, scaling path, and outcome/observability.

Do not merge them into a single oversized note. A development note supports tomorrow's edit; an interview artifact supports concise verbal explanation and follow-up questions.

## Extraction Procedure

1. State the problem in concrete project language, without inflating ordinary code into a highlight.
2. Identify source-confirmed constraints: consistency, duplicate delivery, latency, availability, scale, operational policy, or team-wide reuse.
3. Describe the implemented mechanism and why it matches those constraints.
4. Name one or two viable alternatives and their costs.
5. Cover likely failure or growth scenarios only when relevant: slow/failed storage, duplicate requests/messages, partial failure, cache loss, consumer retry, versioning, traffic growth.
6. Form a 60-90 second narrative using the interview template. Mark business metrics or incident outcomes as unknown unless documented.

## Follow-Up Readiness

For L3/L4 material, prepare answers to: why this boundary exists, why a simpler approach was rejected, where idempotency/transactionality/retry lives, how failures are observed and recovered, what degrades under load, and what would change for a larger scale. Never invent performance numbers, ownership, or incident history.

An artifact is interview-ready only after the user can explain the mechanism, tradeoff, and a changed scenario without relying on the original code.
