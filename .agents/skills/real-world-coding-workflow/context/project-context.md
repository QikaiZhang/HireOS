# Project Context

> Template for repository-specific rules. Inspect it before non-trivial work, but do not treat blanks or examples as rules.

## Discovery Priority

If the repository contains `references/PROJECT_RULES.md`, read it before this file and treat it as the project's primary rules document. Then reconcile it with this context and the actual code. Report conflicts instead of silently choosing one.

## Project Overview

- Project name: [name]
- Purpose: [brief description]
- Main languages and frameworks: [list]

## Repository Structure

Document important directories and ownership boundaries.

```text
cmd/        application entry points
internal/   business implementation
api/        API or Proto definitions
scripts/    project automation
```

## Mandatory Conventions

### MUST

- [Rule]

### SHOULD

- [Rule]

### MAY

- [Rule]

### DO NOT

- [Rule]

## Source Of Truth And Generation

- API/Proto/schema source of truth: [location]
- Generation command: [command]
- Generated files that must not be edited: [patterns]
- Required workflow: source definition -> generator -> generated code -> application code

## Data And Integration Conventions

- Database: architecture, pagination, transactions, and naming.
- Redis: key format, TTL, and permitted data structures.
- MQ/events: broker, naming, retry, idempotency, and consumer rules.
- Plugins: location, loader, build command, and contract.
- Logging: logger, required fields, and redaction rules.
- Errors: error system, codes, wrapping, and caller behavior.

## Build, Test, And Runtime

- Development command: [command]
- Build command: [command]
- Unit test command: [command]
- Integration test command: [command]
- Runtime/deployment assumptions: [details]

## Forbidden Practices

- [For example: direct database access from handlers, hand-editing generated files, or bypassing the repository layer.]

## Known Existing Issues

Known issues are not automatically part of every task. Work on one only when the task depends on it, amplifies it, or explicitly requests it.

| Issue | Trigger | Impact | Origin | Priority | Status |
| --- | --- | --- | --- | --- | --- |
| [Issue] | [Trigger] | [Impact] | Existing | P3 | Known |

## Project-Specific Workflows

Document only real project workflows, for example Proto changes, database migrations, or plugin builds. Include the exact sequence and verification command when known.

## Rule Record

For each material rule, include:

```text
Status: MUST | SHOULD | MAY | DO NOT
Source: file, directory, configuration, or documentation
Reason: why the rule exists
Last Verified: date or revision
```

Add rules only when repeatedly observed, documented, required by the team, or explicitly confirmed. Do not turn a temporary implementation choice into a permanent convention.
