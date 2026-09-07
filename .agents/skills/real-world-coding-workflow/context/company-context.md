# Company Context

> Template for organization-wide engineering conventions shared across repositories. Do not invent policies; mark uncertain rules as `UNKNOWN`.

## Company

- Name: 与你科技

## Local Extism Tooling

For Extism plugin work on the current computer, use this PATH so the project-local Extism binary takes precedence while retaining Homebrew tools:

```sh
PATH="$PWD/bin/extism:/opt/homebrew/bin:$PATH"
```

Rule record:

```text
Status: MUST (for Extism plugin commands on this computer)
Source: user-provided local environment convention
Reason: expose the project-local Extism executable and Homebrew tools
Last Verified: 2026-08-20
```

## Engineering Standards

### MUST

- [Rule, source, and reason]

### SHOULD

- [Rule, source, and reason]

### MAY

- [Rule, source, and reason]

### DO NOT

- [Rule, source, and reason]

## Language Standards

### Go

- [Conventions]

### Python

- [Conventions]

### TypeScript

- [Conventions]

## Shared Standards

- API: naming, versioning, authentication, error response, and pagination.
- Database: migrations, naming, indexes, transactions, and pagination.
- Observability: logger, tracing, metrics, required fields, and sensitive-data restrictions.
- Security: secrets, authorization, data handling, and logging restrictions.
- Testing: required test types, CI checks, and coverage requirements.
- Dependencies: approved or prohibited dependencies and review requirements.
- Deployment: container, Kubernetes, environments, and configuration rules.
- Code review: required checks and review expectations.

## Rule Record

For each important rule, record:

```text
Status: MUST | SHOULD | MAY | DO NOT | UNKNOWN
Source: official policy, repository configuration, or other authoritative reference
Reason: why the rule exists
Last Verified: date or revision
```
