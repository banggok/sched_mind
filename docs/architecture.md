# Architecture Baseline

This document is the reusable entry point for architecture. It deliberately
contains principles and navigation rather than framework configuration,
repository layout, or product behavior.

## Principles

- Organize code around cohesive responsibilities and explicit business
  boundaries.
- Direct dependencies toward the business core. Domain behavior must not depend
  on delivery mechanisms, persistence, frameworks, or external services.
- Keep transport, application workflow, business rules, and infrastructure
  responsibilities separate.
- Apply Domain-Driven Design and SOLID pragmatically. Add a boundary or
  abstraction only when current responsibilities justify it.
- Keep contracts stable across replaceable infrastructure implementations.
- Prefer simple, testable composition over speculative generalization.
- Treat accessibility, operability, failure recovery, security, and
  observability as architectural quality attributes.

## Detailed standards

- [Reusable backend architecture](architecture/backend.md)
- [Reusable frontend architecture](architecture/frontend.md)
- [Reusable frontend design system](frontend-design-system.md)
- [New-project bootstrap workflow](project-bootstrap.md)
- [Project-specific documentation](project/README.md)
- [Current project architecture](project/architecture.md)

## Ownership rule

Reusable documents define deliberate cross-project invariants and decision
criteria. Product rules, framework selections, database dialects, source paths,
runtime topology, and implementation-specific exceptions belong under
`docs/project/` or in authoritative product documents.

A current implementation must not be promoted into this baseline merely because
it exists. Promotion requires deliberate review that the rule remains useful and
truthful when copied into a different product and technology stack.
