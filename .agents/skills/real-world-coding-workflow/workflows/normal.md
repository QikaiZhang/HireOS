# NORMAL Workflow

Use for ordinary feature work: a new endpoint, business flow, cache or consumer, related changes across several files, or a non-local module change.

1. Discover the affected entry point, call chain, data flow, similar implementation, source of truth, and relevant tests.
2. Align the request with applicable company/project context and actual repository conventions.
3. Define the change scope, including files/components deliberately not changing.
4. Make local design decisions and write a brief SPEC when it clarifies responsibilities, interfaces, or verification. Use [../templates/spec.md](../templates/spec.md).
5. Analyze material risks with [../templates/risk-analysis.md](../templates/risk-analysis.md).
6. Implement the minimal verifiable slice: input, core logic, persistence/external effect, output, and proof of the required outcome.
7. Verify new behavior and relevant existing behavior with [../templates/verification.md](../templates/verification.md).
8. Review scope, unresolved risks, generated-code workflow, and regression exposure before reporting completion.

Pause only for high-impact choices. Do not ask the user to approve ordinary code structure, variable names, helper methods, local errors, or straightforward tests.
