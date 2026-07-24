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

# User Experience Rules

User experience is a product requirement, not only a presentation concern.

Frontend implementation must consider usability, clarity, consistency,
accessibility, feedback, and failure recovery.

## Core UX Principles

- Prefer clear and predictable behavior over clever interaction.
- Minimize unnecessary user effort.
- Preserve user input whenever possible.
- Make system status visible.
- Prevent avoidable errors before they occur.
- Provide a clear recovery path when errors happen.
- Keep terminology consistent with the business domain.
- Do not sacrifice usability merely to simplify implementation.
- Do not introduce interaction complexity without a confirmed user need.

## Interaction Feedback

Every user-triggered asynchronous action must provide appropriate feedback.

Examples:

- Loading state
- Disabled state
- Success confirmation
- Error message
- Empty state
- Progress indication for long-running operations

Rules:

- Do not allow repeated submission while the same action is still processing.
- Disable only the controls affected by the current operation.
- Do not block the entire page when only one section is loading.
- Avoid loading indicators that cause unnecessary layout shifts.
- Do not show a success message before the operation is confirmed.
- Do not leave users uncertain whether an action succeeded.

## Loading States

- Use loading indicators appropriate to the expected duration and affected area.
- Prefer local loading states over full-page loading states.
- Preserve existing visible data during background refresh when safe.
- Avoid replacing useful content with an empty spinner during refetch.
- Use skeletons only when they improve perceived continuity.
- Do not show a loading indicator for synchronous or near-instant interactions
  unless needed to prevent duplicate actions.

## Error Handling and Recovery

User-facing errors must:

- Explain what failed in understandable language.
- Avoid exposing stack traces, internal codes, database details, or
  infrastructure details.
- Indicate whether the user can retry.
- Preserve user input when retry is possible.
- Provide the next reasonable action.

Rules:

- Do not use generic messages such as "Something went wrong" when a more useful
  explanation is available.
- Do not expose raw backend error messages directly.
- Map technical errors into user-understandable messages at the presentation
  boundary.
- Field validation errors should appear near the affected field.
- Page-level or operation-level errors should appear near the relevant action or
  content.
- Use toast notifications only for transient, non-blocking feedback.
- Do not use toast notifications as the only place for critical errors.
- Failed optimistic updates must be reverted or clearly reconciled.

## Form UX

- Clearly identify required fields.
- Use labels, not placeholders, as the primary field description.
- Preserve entered values after validation or submission failure.
- Validate as early as useful, but do not interrupt users excessively.
- Show actionable validation messages.
- Focus or scroll to the first invalid field when appropriate.
- Do not submit invalid forms.
- Prevent duplicate submission.
- Do not silently transform user input unless the transformation is obvious and
  safe.
- Confirm destructive or irreversible actions.
- Avoid confirmation dialogs for safe and easily reversible actions.

## Data Presentation

- Present the most important information first.
- Use business terminology consistently.
- Use human-readable dates, durations, statuses, and quantities.
- Do not rely only on color to communicate status.
- Distinguish loading, empty, error, and successful zero-result states.
- Tables and lists should clearly communicate sorting, filtering, pagination,
  and selection state.
- Preserve filters, sorting, pagination, and draft changes when users navigate
  within the same workflow when practical.
- Avoid displaying raw IDs when a meaningful business label is available.
- Large datasets must not be rendered without appropriate pagination,
  virtualization, or incremental loading when required.

## Empty States

An empty state must distinguish between:

- No data exists yet.
- No results match the current filter.
- Data failed to load.
- The user does not have permission.
- The feature is unavailable.

Whenever appropriate, provide a clear next action.

## Navigation and Workflow

- Navigation must be predictable.
- The current location and active state should be visible.
- Users should not lose unsaved work without warning.
- Multi-step workflows should show current progress when the sequence is not
  obvious.
- Back navigation should not unexpectedly reset completed work.
- Avoid forcing users through unnecessary screens or confirmations.
- Keep primary actions easy to identify.
- Limit competing primary actions on the same view.

## Destructive Actions

For delete, reset, revoke, overwrite, or other destructive actions:

- Clearly state what will be affected.
- Require confirmation when the action is difficult or impossible to reverse.
- Use specific confirmation language rather than generic "Are you sure?"
- Prefer undo when the action is easily reversible.
- Do not visually emphasize destructive actions as the default primary action.
- Do not place destructive actions where they can be triggered accidentally.

## Accessibility

Frontend code should meet practical accessibility standards.

Requirements:

- Use semantic HTML.
- Ensure interactive elements are keyboard accessible.
- Provide visible focus states.
- Associate labels with form controls.
- Provide meaningful accessible names for icon-only controls.
- Use ARIA only when native semantic HTML is insufficient.
- Maintain sufficient contrast.
- Do not communicate meaning using color alone.
- Support reasonable text resizing.
- Respect reduced-motion preferences where animation is used.
- Ensure modals manage focus correctly.
- Ensure validation and dynamic status messages can be understood by assistive
  technology.

## Responsive Design

- Core workflows must remain usable on supported screen sizes.
- Do not hide required actions on smaller screens.
- Avoid horizontal scrolling for normal page content.
- Tables may use responsive alternatives when the full table cannot remain
  usable.
- Touch targets must be reasonably sized and spaced.
- Do not assume hover is available.
- Responsive behavior must be tested, not inferred only from CSS classes.

## Performance and Perceived Performance

- Avoid unnecessary network requests and re-renders.
- Do not fetch data that is not required for the current view.
- Avoid blocking the initial interface with non-critical data.
- Use caching only when its invalidation behavior is understood.
- Avoid premature optimization that adds architectural complexity.
- Prioritize perceived responsiveness for common user actions.
- Debounce or throttle interactions only when it improves behavior and does not
  hide user intent.

## State Consistency

- The interface must reflect the confirmed system state.
- Temporary UI state and persisted server state must be distinguished.
- Unsaved changes must be visibly identifiable.
- After mutation, update or invalidate affected data consistently.
- Do not leave stale data visible after a confirmed change.
- Do not silently discard local changes.
- Optimistic updates should only be used when rollback and reconciliation are
  defined.

## Permission and Availability

- Do not display actions users cannot perform unless there is a clear UX reason.
- UI restrictions are not a substitute for backend authorization.
- Permission errors must be handled explicitly.
- Disabled actions should explain why they are unavailable when the reason is
  not obvious.
- Feature availability and permission state must not be inferred only from
  hidden UI elements.

## Consistency

- Reuse existing interaction patterns and shared UI components.
- Similar actions should behave similarly.
- Status labels, button terminology, date formats, spacing, and validation
  behavior must remain consistent.
- Do not create a new interaction pattern when an established project pattern
  already solves the problem.
- Shared components must not hide important business behavior behind overly
  generic APIs.

## UX Testing

Tests should cover important observable user behavior, including:

- Loading state
- Success state
- Empty state
- Validation failure
- Backend failure
- Retry behavior
- Duplicate submission prevention
- Unsaved-change protection when applicable
- Permission-restricted behavior
- Keyboard interaction for critical workflows
- Responsive behavior when materially different

Do not:

- Test implementation details such as internal hook state when user-visible
  behavior can be tested instead.
- Rely only on snapshots for important interactions.
- Remove accessibility attributes merely to make tests easier.

## UX Review During Self Review

During self-review, verify:

- Is the primary action obvious?
- Does every asynchronous action provide feedback?
- Can the user recover from failure?
- Is user input preserved?
- Are loading, empty, success, and error states distinct?
- Are destructive actions sufficiently protected?
- Are permission and disabled states understandable?
- Is the workflow keyboard accessible?
- Is the interface usable on supported screen sizes?
- Does the implementation introduce unnecessary interaction complexity?
- Does the behavior use consistent business terminology?

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

# Integration Configuration

- Configuration for databases, external APIs, queues, caches, storage, and
  other third-party integrations must come from environment variables.
- Never hardcode integration URLs, hosts, ports, credentials, tokens, API keys,
  or environment-specific identifiers in application source code.
- Local, test, staging, and production environments must be configurable
  without changing source code.
- Document required variables in `.env.example` using safe development values
  or placeholders.
- Never commit secrets or real production credentials.
- Fail fast with a clear error when required integration configuration is
  missing or invalid.
- Tests must inject their own configuration and must not depend on developer or
  production environment variables.

---

# Error Handling

- Domain errors represent business failures.
- Infrastructure errors remain in infrastructure.
- Map errors to HTTP only in transport.
- Preserve error causes.
- Never expose internal implementation details.
- When a Go function returns an object or struct together with an error, and
  that result is invalid whenever the error is non-nil, return a pointer and
  use `nil, err` on every failure path.
- Never return a zero-value entity, DTO, configuration, or other object beside
  a non-nil error when callers could accidentally use that invalid value.
- Slices, maps, interfaces, and existing pointer results must be nil on error
  unless a documented partial-result contract explicitly requires otherwise.
- Do not introduce pointer wrappers for booleans or scalar values when their
  zero value is an intentional and unambiguous part of the contract.
- Always check a returned pointer for nil before dereferencing it or accessing
  its fields, even after confirming that the returned error is nil.
- Treat `nil, nil` from a dependency that promises a result as a contract
  violation and return a clear internal error instead of panicking or
  continuing with incomplete state.

---

# Process Lifecycle

- Long-running services must implement graceful shutdown.
- Handle both interrupt and termination signals where the operating system
  supports them.
- Stop accepting new work before releasing infrastructure resources.
- Allow in-flight work to finish within a configurable timeout.
- Close database connections, consumers, queues, and other owned resources
  during shutdown.
- Preserve and report startup, runtime, shutdown, and cleanup errors.
- Do not wait indefinitely during shutdown.

---

# Database Queries and Indexes

- Design every production query together with the index strategy that supports
  its filters, joins, uniqueness checks, and ordering.
- Do not introduce a query that unintentionally forces a full table scan when
  an appropriate index can support the access pattern.
- Consider index behavior and syntax for both PostgreSQL and MySQL when
  designing queries, even when only one dialect is currently enabled.
- When PostgreSQL and MySQL require different index definitions, keep the
  application and domain unchanged and provide dialect-specific infrastructure
  migrations.
- Document the minimum database version when relying on features such as
  functional indexes, generated columns, or specialized collations.
- Verify non-trivial query plans with the dialect's `EXPLAIN` tooling when that
  dialect is supported by the project.
- Use GORM query and persistence APIs for runtime data access. Do not use
  handwritten raw `SELECT`, `INSERT`, `UPDATE`, or `DELETE` statements in
  runtime repositories or application code.
- Raw SQL is permitted only for version-controlled schema migrations and
  isolated automated-test schema setup when the ORM cannot express the
  required DDL, constraints, or indexes.
- Keep permitted raw migration and test-schema SQL inside infrastructure
  boundaries, parameterize values where applicable, and never construct it
  from untrusted input.
- Do not add indexes mechanically. Consider write cost, storage, selectivity,
  duplicate indexes, and the actual query access pattern.

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
