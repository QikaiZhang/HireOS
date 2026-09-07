# FAST Workflow

Use for low-risk, local work such as a small bug fix, obvious parameter change, focused test addition, simple SQL adjustment, formatting, or local refactor.

1. Inspect the affected code, relevant callers, and applicable project context.
2. Identify the smallest change that satisfies the request.
3. Implement it in the local style.
4. Run proportionate verification.
5. Report the change, verification performed, and anything not verified.

Do not require a formal SPEC, risk table, or confirmation for ordinary implementation decisions. Switch to NORMAL or DEEP before changing an API or Proto contract, schema, external behavior, concurrency or security-sensitive path, generated source, or a larger-than-expected scope.
