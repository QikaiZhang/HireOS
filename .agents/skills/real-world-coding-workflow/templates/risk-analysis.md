# Risk Analysis

Identify issues relevant to the current change, rather than every issue in the codebase.

## Classification

- **Existing**: present before this work and not made more likely or severe by it.
- **Introduced**: created by this change.
- **Amplified**: already present, but made more likely or more severe by this change.

## Priority

- **P0**: critical data loss, severe security failure, broken core functionality, or unrecoverable consistency failure. Resolve immediately.
- **P1**: the change cannot safely ship without a fix. Resolve in this task.
- **P2**: important, directly related reliability, regression, or scalability concern. Fix when contained; otherwise document why it is deferred.
- **P3**: historical or unrelated issue. Record when useful; do not expand scope automatically.

## Analysis Table

| ID | Problem | Trigger | Origin | Current Feature Impact | Test Impact | Data Impact | Production Impact | Priority | Action |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| R1 | [Problem] | [Trigger] | Existing | No | No | No | Low | P3 | Record |
| R2 | [Problem] | [Trigger] | Introduced | Yes | Yes | No | High | P1 | Fix |
| R3 | [Problem] | [Trigger] | Amplified | Yes | No | No | Medium | P2 | Evaluate |

## Required Reasoning For P0-P2

1. What fails and what triggers it?
2. What evidence establishes whether it is Existing, Introduced, or Amplified?
3. How does it affect current behavior, tests, data correctness, and production?
4. Why is this priority appropriate?
5. Why must it be fixed now, or why is deferral acceptable?

Reclassify an Existing issue as Amplified when the current change makes it more likely or severe.
