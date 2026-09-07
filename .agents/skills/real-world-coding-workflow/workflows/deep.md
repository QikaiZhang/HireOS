# DEEP Workflow

Use for architecture changes, schema and data migrations, major Proto changes, distributed consistency, concurrency, performance-critical code, security-sensitive behavior, large dependencies, major refactors, or cross-service work.

1. Perform focused but comprehensive discovery: architecture, dependencies, callers, data ownership, operational assumptions, existing tests, and source-of-truth workflows.
2. Align requirements with applicable company/project rules and report documentation-versus-code conflicts.
3. Define the change boundary and impact across APIs, Proto, databases, Redis, MQ, callers, deployment, observability, and tests as relevant.
4. Compare meaningful options, recording reversibility, compatibility, operational cost, and failure modes.
5. Complete [../templates/risk-analysis.md](../templates/risk-analysis.md) and [../templates/spec.md](../templates/spec.md), including compatibility and rollback considerations. For backward-incompatible or multi-service contracts, include consumer sequencing, versioning or compatibility windows, deprecation, rollout, and rollback.
6. Present the selected high-impact decisions and wait for user confirmation before the irreversible or contract-affecting change.
7. Implement in incremental, testable slices. Re-evaluate scope and risks whenever discovery invalidates the plan.
8. Verify the new behavior, regression behavior, and relevant operational assumptions using [../templates/verification.md](../templates/verification.md).
9. Report remaining risks, deferred work, and anything that could not be verified.

Do not continue past the decision gate based on implied approval. The confirmation is for the consequential design or irreversible action, not for trivial implementation details.
