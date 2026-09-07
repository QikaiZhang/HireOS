# SPEC: [Feature Name]

## Background

Current behavior and the problem being solved.

## Goal

Required outcome.

## Non-Goals

Explicitly out-of-scope work, such as unrelated refactoring, historical cleanup, speculative architecture, or performance work.

## Existing Implementation

- Entry point: [location]
- Call chain:

```text
Input -> validation -> business logic -> persistence or external system -> output
```

- Similar implementation: [reference]
- Relevant project conventions: [company context, project context, code references]

## Proposed Change

Describe responsibilities, data flow, interfaces, and material implementation decisions. Avoid prematurely inventing a large list of functions.

## Change Scope

### Modify

- [file or component]

### Add

- [file or component]

### Do Not Modify

- [file or component]

## Key Design Decisions

| Decision | Selected Approach | Reason | Alternative | Impact | Confirmation Needed |
| --- | --- | --- | --- | --- | --- |
| [Decision] | [Approach] | [Reason] | [Alternative] | [Impact] | [Yes or No] |

## Minimal Verifiable Slice

```text
Input -> core logic -> persistence or external effect -> output -> verification
```

## Risks And Compatibility

Use [risk-analysis.md](risk-analysis.md). Analyze API, Proto, database, Redis, MQ, existing callers, tests, and rollback where relevant.

## Verification

List build, unit, integration, manual, and regression checks using [verification.md](verification.md).

## Acceptance Criteria

- [ ] Required behavior works.
- [ ] Relevant existing behavior is preserved.
- [ ] P0 and P1 risks are resolved.
- [ ] Remaining P2/P3 risks are documented.
- [ ] Change scope is respected.
