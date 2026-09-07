# Source Reading Rules

## Evidence And Scope

Use three labels whenever a conclusion could influence implementation or an interview narrative:

- **Confirmed:** Directly evidenced by source, configuration, tests, or a trace.
- **Inferred:** Plausible from multiple local signals but not fully shown.
- **Needs confirmation:** Depends on missing code, runtime behavior, schema, deployment configuration, or an external service contract.

Never infer a complete business process from class, package, or method names. State the missing evidence and inspect adjacent callers, tests, configuration, interfaces, or implementations before asking the user for more input.

## Module Map

For a module-level question, collect only the fields needed to answer it. Do not display a full map for a local question.

| Field | What to identify |
| --- | --- |
| Responsibility | Business problem and ownership boundary. |
| Layer | Entry/transport, application/service, domain, repository/client, worker, or infrastructure. |
| Upstream | Concrete routes, consumers, jobs, callers, or interfaces. |
| Downstream | Storage, remote clients, queues, caches, internal libraries, and side effects. |
| Contract | Inputs, outputs, validations, error/result conventions, and state transitions. |
| Operational behavior | Transactions, retries, idempotency, logging, metrics, tracing, and asynchronous work. |

## Code Classification

- **A: Core business code.** Rules, orchestration, state transitions, pricing/permissions, invariants. Read deeply.
- **B: Engineering infrastructure.** Transactions, caching, queues, resilience, observability, internal abstractions. Read deeply when it controls the path.
- **C: Framework glue.** Controllers, wiring, annotations/decorators, registration, adapters. Read enough to establish boundaries.
- **D: Mechanical code.** DTO mapping, boilerplate, repeated accessors, generated code. Summarize unless it changes semantics.

## Dependencies

Classify dependencies before setting depth:

- **Business core:** Trace its real callers and invariants deeply.
- **Internal library:** Explain why it is wrapped or standardized, its shared capability, contract, constraints, extension point, and why business code relies on it. Avoid line-by-line implementation unless that implementation is the problem.
- **Third-party SDK:** Explain capability, local usage, important configuration/lifecycle, and error behavior. Do not enter vendor internals without an explicit request.
- **Standard library:** Explain the required API semantics and any mechanism that changes correctness; avoid general textbook exposition.

When a likely unfamiliar term appears, define it in three short parts: what it is, why this code needs it, and its local role. Start with the local relationship before a general definition. Do not repeat an already-established concept.

## Large Inputs

For a service, directory, or many files: find manifests/configuration, process entry points, transports, core application services, persistence/remote boundaries, and tests. Give a small map, select one path that serves the user's question, explain why it is the right start, and trace it. Do not continue to a second path until the user asks. Prefer entry point, central business flow, critical data flow, and infrastructure controls over edge features and templates.

For a requested change, work backward from the target behavior and forward from the entry point until the paths meet. Identify contract changes, data migrations, event consumers/producers, retries/idempotency effects, authorization, and tests before recommending edits.
