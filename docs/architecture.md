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

## Testing Boundaries

Testing follows the same architectural boundaries as production code.

### Domain

Domain tests prove entities, value objects, invariants, and domain services without transport, persistence, or framework dependencies.

### Application

Application tests prove use-case orchestration, repository interaction, authorization decisions, and transaction intent. Repository interfaces may be replaced with controlled fakes where persistence semantics are not under test.

### Infrastructure

Infrastructure integration tests use a real supported test database or equivalent infrastructure boundary. They prove:

- schema constraints;
- repository queries;
- relation integrity;
- transaction commit and rollback;
- locking and concurrency behaviour;
- persistence mapping.

Repository integration tests must not be replaced by mocks when the Acceptance Criterion depends on persisted state.

### Transport

HTTP acceptance tests execute requests through the actual router and handler stack. They verify:

- request validation;
- HTTP status;
- stable error code;
- response body;
- externally observable persistence effects.

Calling a handler function directly without the real routing and serialization boundary is a transport unit test, not an HTTP acceptance test.

### Frontend

Frontend component tests prove isolated rendering and interaction behaviour.

Frontend acceptance-level feature tests must render the feature through its public composition boundary and exercise the actor workflow using user-visible controls. They must verify visible results and gateway interactions rather than component internals.

### End-to-End

End-to-end tests are required only when an Acceptance Criterion cannot be adequately proven through backend HTTP acceptance tests and frontend feature acceptance tests independently.

Do not create end-to-end tests for every AC by default. Prefer the highest practical boundary that provides deterministic and maintainable evidence.
