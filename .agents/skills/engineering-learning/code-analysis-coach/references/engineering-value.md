# Engineering Value And Retention

## Value Levels

| Level | Meaning | Typical material | Default action |
| --- | --- | --- | --- |
| L1 | Local or mechanical knowledge. | CRUD, ordinary validation, DTO conversion, simple controller. | Explain briefly; do not retain by default. |
| L2 | Reusable engineering practice. | Cache, transaction, retry, idempotency, rate limit, queue, connection pool. | Proactively explain one reusable insight when source shows a real decision; retain only on request. |
| L3 | Architecture or system design. | Service boundaries, state machine, event flow, consistency scheme, failure recovery, high-concurrency control. | Proactively explain the design boundary and its local reason; create notes or interview material only on request. |
| L4 | Defensible project highlight. | Complex problem with explicit tradeoffs across performance, reliability, consistency, or scale. | Explain the transferable tradeoff; create a focused interview artifact only on request. |

## What Makes Code Worth Deep Reading

Prioritize code that controls a business invariant, coordinates multiple dependencies, defines a failure boundary, changes persistent state, makes an asynchronous handoff, encodes an operational policy, or is broadly reused through an internal abstraction.

Do not create value merely because a familiar technology name appears. Redis, a message queue, or a transaction becomes L2+ only when the source reveals why it is used, how it is configured/handled, or a meaningful consequence. A design becomes L4 only when its problem, constraints, tradeoffs, and transferable reasoning can be explained from evidence.

## Engineering Questions

For an L2+ item, answer only questions supported by the material:

- What correctness, latency, throughput, cost, reliability, or team-scale problem does it solve?
- Why does it live at this layer and own this responsibility?
- What happens if it is removed, bypassed, or fails?
- What alternative is plausible here, and what does the current choice trade away?
- What observation, test, or operational safeguard makes the behavior trustworthy?

Use cross-framework comparisons only when they clarify a non-obvious local decision. Compare the responsibility and tradeoff, not superficial syntax across Spring Boot, Go, go-zero, Gin, NestJS, or a standard library.

## Default Delivery

During ordinary code reading, express one supported L2+ insight as a short, local "可以带走的一点". It must identify the local evidence and a reusable decision rule. Do not show the L-level, assign homework, or start a persistence flow. If the code only shows that a technology exists, not why or how its behavior matters, do not manufacture a lesson.
