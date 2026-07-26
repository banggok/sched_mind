# Reusable Backend Architecture

This document defines technology-neutral backend boundaries. A project's actual
language, framework, database, directory names, and exceptions belong in its
project-specific architecture document.

## Dependency direction

Dependencies point toward the business core:

```text
transport -> application -> domain
infrastructure -> application/domain contracts
```

The composition root may depend on all outer layers to assemble the application.
Inner layers must not import outer layers.

## Domain

The domain owns entities, value objects, invariants, domain services, and domain
errors. It must not depend on transport protocols, persistence models, database
clients, external-service SDKs, or application frameworks.

Enforce business invariants where invalid domain state is created or changed.
Use a value object when a value has meaningful validation or behavior. Use a
domain service only when behavior does not naturally belong to an entity or
value object. Do not introduce aggregates, factories, or domain events without
a demonstrated business need.

## Application

The application layer coordinates use cases, workflows, authorization decisions
that depend on the use case, transaction boundaries, and calls to ports. It may
combine domain operations but must not implement HTTP mapping, database access,
framework behavior, or presentation concerns.

Repository and integration interfaces belong near the application consumer when
they represent capabilities required by a use case. Prefer small,
consumer-oriented ports; do not create an interface for every concrete type.

Transactions should cover the smallest complete business operation. The
application layer owns the need for atomicity, while infrastructure supplies the
transaction mechanism. Define concurrency behavior when simultaneous operations
can violate an invariant.

## Transport

Transport adapters receive protocol input, validate its shape, map it to
application input, invoke a use case, and map output or errors back to the
protocol. They may handle HTTP, RPC, messages, scheduled jobs, or CLI commands.

Transport validation protects the protocol contract; it does not replace domain
validation. Handlers must not contain business decisions, persistence queries,
or infrastructure-specific failure handling.

## Infrastructure

Infrastructure implements persistence, caches, queues, storage, external APIs,
clock or identity adapters, and application ports. It owns framework setup,
connection lifecycle, DTO and persistence-model mapping, migrations, and
technology-specific errors. It must not define business rules.

Keep environment-specific integration values outside source code. Validate
required configuration at startup and inject test configuration explicitly.

## Models and mapping

Keep domain entities, transport DTOs, and persistence models distinct when they
have different responsibilities or change for different reasons. Map at the
boundary that owns the external representation. Never expose persistence models
as domain or API models by convenience.

## Errors and validation

- Domain errors describe business failures without infrastructure detail.
- Application errors describe use-case failures or violated dependency
  contracts.
- Infrastructure errors remain internal and preserve their causes.
- Transport maps known errors to stable protocol responses and safe messages.
- Do not return an invalid object alongside a non-nil error when callers could
  use it accidentally.

Validate syntax at the transport boundary, invariants in the domain, and
database constraints in persistence as defense in depth. Database constraints
must not be the only expression of a business invariant.

## Persistence and queries

- Design each production query with its filters, joins, deterministic ordering,
  expected cardinality, and supporting indexes.
- Every new or changed production query requires an explicit review before
  completion. Review equality and range predicates, joins, soft-delete or tenant
  scopes added by frameworks, ordering, pagination, expected result volume, and
  concurrency or locking behavior. Confirm whether an existing index supports
  the effective query rather than only its handwritten predicate.
- Add or change an index when the reviewed access pattern justifies it. Record
  why no new index is needed when a primary key, unique constraint, existing
  composite index, bounded cardinality, or another demonstrated access path is
  sufficient.
- Evaluate index selectivity, write cost, storage cost, and supported database
  behavior rather than adding indexes mechanically.
- Keep dialect-specific DDL and optimizations in infrastructure without changing
  domain or application contracts.
- Keep schema changes version-controlled and reversible where practical.
- Avoid unbounded production reads; use pagination, streaming, or an explicitly
  bounded dataset.
- Verify non-trivial access paths with the supported database's query-plan tools
  against representative cardinality when practical. Treat a sequential scan on
  a tiny table as a planner cost decision, not automatically as a defect; verify
  the available indexed path separately when needed.
- Include query-review findings, indexes added or removed, plan verification,
  and remaining cardinality assumptions in the completion report.

Whether runtime raw SQL or a particular ORM is allowed is a project-specific
persistence decision, not a universal architecture rule.

## Process lifecycle

Long-running processes must stop accepting new work, allow in-flight work to
finish within a bounded timeout, release owned resources, and report startup,
runtime, shutdown, and cleanup failures. Handle the termination signals
supported by the runtime environment and never wait indefinitely.

## Testing boundaries

- Domain tests cover invariants, value behavior, boundaries, and domain errors
  without framework or database dependencies.
- Application tests cover orchestration, dependency failure, atomicity intent,
  and business outcomes through ports or test doubles.
- Infrastructure integration tests cover mappings, constraints, transactions,
  concurrency, query behavior, and the real adapter contract.
- Transport integration tests cover routing, request and response mapping,
  validation, status or error mapping, and persistence effects when applicable.
- Composition and smoke tests verify that configured adapters work together.

Prefer observable behavior over implementation details. Cover success,
validation failure, business failure, dependency failure, boundary values,
rollback, and concurrency when each is applicable.
