# AGENTS.md

# Project Overview

This project follows Domain-Driven Design (DDD) and applies SOLID principles pragmatically.

The objective is to maintain clear business boundaries, low coupling, high cohesion, and testable components without introducing unnecessary abstractions.

---

# Architecture

Use the existing project architecture and directory structure as the primary reference.

Typical responsibilities:

## domain

- Business entities
- Value Objects
- Domain Services
- Domain Errors

Must NOT depend on:

- HTTP
- Database
- Framework
- Infrastructure

---

## application / usecase

Responsible for:

- Business workflows
- Use case orchestration
- Transaction coordination

Must NOT contain:

- HTTP logic
- Database logic
- Framework logic

---

## transport / handler

Responsible for:

- HTTP
- gRPC
- Message consumer
- CLI

Responsibilities:

- Validation
- Request mapping
- Response mapping

Must NOT contain business logic.

---

## infrastructure

Responsible for:

- Repository implementation
- Database
- Cache
- Queue
- External API
- Storage

Must NOT contain business rules.

---

# Frontend Architecture

The frontend uses feature-oriented Clean Architecture pragmatically.

Do not create layers or abstractions before an actual feature requires them.

For business features, prefer this structure:

```text
src/
├── app/
├── features/
│   └── <feature-name>/
│       ├── domain/
│       ├── application/
│       ├── infrastructure/
│       └── presentation/
└── shared/
```

## Frontend Layer Responsibilities

### domain

Responsible for:

- Business models
- Business rules
- Value Objects
- Domain validation
- Domain errors

Must NOT depend on:

- React
- HTTP clients
- Browser APIs
- State-management libraries
- UI libraries
- API DTOs

### application

Responsible for:

- Use case orchestration
- Application workflows
- Ports consumed by use cases
- Mapping application results

Must NOT contain:

- React components
- HTTP implementation
- Styling
- Browser-specific behavior

### infrastructure

Responsible for:

- HTTP API clients
- Storage adapters
- External service integration
- API DTO mapping
- Implementations of application ports

Must NOT contain business rules.

### presentation

Responsible for:

- React components
- Pages
- Hooks
- View models
- User interaction
- Presentation state

Must NOT:

- Implement business rules
- Use raw API DTOs as domain models
- Call HTTP clients directly when a use case exists
- Contain persistence or integration logic

## Frontend Dependency Direction

Dependencies must point toward the business core:

```text
presentation → application → domain
infrastructure → application/domain contracts
```

The domain layer must remain framework-independent.

## Frontend Implementation Rules

- Organize business code by feature.
- Keep feature-specific code inside its feature boundary.
- Put code in `shared` only when it is genuinely reused by multiple features.
- Do not create generic repositories, services, factories, or hooks without an actual need.
- Separate API DTOs from domain models when their responsibilities differ.
- Map API data at the infrastructure boundary.
- Keep React components focused on rendering and interaction.
- Move business decisions out of components and hooks into domain or application code.
- Do not introduce interfaces for every function or service.
- Prefer plain functions and TypeScript types until polymorphism or substitution is required.
- Preserve observable behavior during refactoring.
- Do not refactor unrelated code while implementing a feature.

---

# Domain-Driven Design Rules

- Business logic belongs in the Domain layer.
- Use business terminology consistently.
- Preserve bounded contexts.
- Do not expose persistence models as domain models.
- Separate Entity, DTO, and Database Model when responsibilities differ.
- Enforce business invariants inside the Domain.
- Use Value Objects when validation or behavior exists.
- Use Domain Services only when behavior does not naturally belong to one Entity.
- Repository interfaces should be defined near the consuming layer.
- Do not introduce Aggregates, Factories, Domain Events, or additional layers unless the business truly requires them.

---

# SOLID Principles

Apply SOLID pragmatically.

## SRP

- One reason to change.
- Separate transport, business, and infrastructure.
- Do not split classes/packages only to satisfy SRP mechanically.

## OCP

Prefer extension over modification only when multiple behaviors actually exist.

Do not create extension points for hypothetical future requirements.

## LSP

Implementations must preserve behavioral contracts.

## ISP

Prefer consumer-oriented interfaces.

Do not create interfaces merely because "SOLID says so."

## DIP

Core business depends on abstractions only at meaningful architectural boundaries.

Do not wrap every concrete implementation with an interface.

---

# Implementation Rules

- Preserve the current architecture.
- Keep changes minimal.
- Reuse existing components.
- Avoid unrelated refactoring.
- Do not introduce new dependencies without approval.
- Do not change public contracts unless requested.
- Never manually modify generated code.
- Avoid circular dependencies.
- Avoid generic helper packages containing business logic.
- Prefer composition over inheritance.
- Simplicity is preferred over cleverness.

---

# Error Handling

- Domain errors represent business failures.
- Infrastructure errors remain in infrastructure.
- Map errors to HTTP only in transport.
- Preserve error causes.
- Never expose internal implementation details.

---

# Testing Standards

Tests must verify behavior rather than implementation.

Cover:

- Success case
- Validation failure
- Business rule violation
- Dependency failure
- Edge case
- Zero value
- Boundary case
- Concurrency when applicable

Do not:

- Reduce assertions merely to make tests pass.
- Generate meaningless tests for coverage.

---

# Validation

Run relevant validation after implementation.

Go:

go fmt ./...
go vet ./...
go test ./...
go test -race ./...

Frontend:

npm test
npm run lint
npm run build

Fix failures before completion.

---

# Required Workflow

## Phase 1 — Understand

Before modifying code:

- Understand the business requirement.
- Inspect relevant modules.
- Identify affected files.
- Identify architectural boundaries.
- State assumptions.
- Ask for clarification if the requirement is ambiguous.

---

## Phase 2 — Planning

For non-trivial changes:

- Propose implementation approach.
- List affected files.
- Explain why each file changes.
- Identify risks.
- Keep scope minimal.

Do not implement until the approach is clear.

---

## Phase 3 — Implementation

Implement only the approved scope.

Requirements:

- Follow DDD.
- Apply SOLID pragmatically.
- Keep changes minimal.
- Avoid unrelated refactoring.
- Reuse existing code.
- Avoid unnecessary abstractions.

---

## Phase 4 — Validation

Run all relevant validation commands.

Fix failures before proceeding.

---

## Phase 5 — Self Review

Review implementation critically.

Verify:

- DDD boundary
- SOLID adherence
- Error handling
- Concurrency
- Security
- Performance
- Edge cases
- Test quality
- Overengineering
- Unnecessary abstractions

Fix confirmed issues.

---

## Phase 6 — Completion Report

Always summarize:

- Business requirement implemented
- Files modified
- Architectural approach
- Tests added or updated
- Validation results
- Remaining risks
- Assumptions
- Technical debt introduced (if any)

---

# Completion Output

Include:

1. Summary
2. Root cause (for bug fixes)
3. Business behavior affected
4. Files changed
5. Architectural decisions
6. Tests executed
7. Validation results
8. Remaining risks
9. Follow-up recommendations (if applicable)
