# DEBUGGING Workflow

Begin with evidence, not a code change.

1. Reproduce the failure or establish why it cannot yet be reproduced.
2. Observe the symptom through tests, logs, metrics, traces, inputs, outputs, or a minimal diagnostic.
3. Trace the affected control and data flow.
4. State competing hypotheses with supporting and contradicting evidence.
5. Choose the cheapest, highest-information check to verify or reject each important hypothesis.
6. Identify the root cause, distinguishing it from downstream symptoms.
7. Design and implement the smallest fix that removes the cause.
8. Add or run a regression test that fails without the fix when practical.

If the failure cannot be reliably reproduced, capture production-safe diagnostics, state the evidence still needed, and define a clear hand-off or stop condition rather than claiming a root cause.

Do not increase timeouts, retries, buffers, concurrency, or resource limits merely to hide a symptom without explaining the root cause and tradeoff. Escalate to NORMAL or DEEP when the root cause has non-local, contract, data, concurrency, or security impact.
