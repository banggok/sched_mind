# Reusable Frontend Architecture

This document defines framework-neutral frontend boundaries. Frameworks, build
tools, route libraries, state libraries, and source paths are project-specific
unless deliberately adopted as reusable constraints.

## Dependency direction

For business features, dependencies point toward the business core:

```text
presentation -> application -> domain
infrastructure -> application/domain contracts
app composition -> feature entry points and shared primitives
```

The domain must remain independent of UI frameworks, browser APIs, API DTOs,
HTTP clients, storage libraries, and state-management libraries.

## Application shell, routes, and pages

The application boundary owns bootstrap, route composition, global providers,
navigation metadata, and persistent application chrome. Feature pages render
inside that shell and must not create competing top bars, sidebars, routers, or
global providers.

Routes identify workflows and compose feature entry points. A route or page may
coordinate presentation state, but business decisions belong in domain or
application code. Navigation behavior such as scroll restoration should have a
single owner rather than being duplicated by pages.

## Feature modules

Organize business code by feature. Introduce only the layers justified by real
responsibilities:

- `domain`: models, value behavior, invariants, validation, and domain errors.
- `application`: use-case orchestration, workflows, ports, and application
  result mapping.
- `infrastructure`: HTTP clients, storage adapters, external integrations, API
  DTO mapping, and implementations of application ports.
- `presentation`: pages, components, hooks, view models, interaction, and
  presentation state.

Presentation must not call infrastructure directly when an application use case
exists, implement business invariants, persist data, or treat a raw API DTO as a
domain model. Infrastructure must map external data at its boundary and must not
contain business rules.

Do not create empty layer folders. Plain functions and language types are
preferred until polymorphism or substitution is needed.

## Type safety

Project-owned TypeScript source and tests must not use explicit or implicit
`any`. Model known data precisely. External JSON, browser messages, storage
values, third-party callbacks without a trustworthy contract, and other dynamic
inputs enter the application as `unknown` and must be narrowed or validated at
their boundary before reaching application or domain code.

Do not replace `any` with an unsafe assertion, a lint suppression, or a generic
that merely hides the same uncertainty. Generated sources and third-party type
declarations are governed by their producer and are not manually modified.

## Shared code and UI primitives

Place code in `shared` only after it is genuinely reused or represents a stable
application-wide primitive. Shared code must have a narrow responsibility and
must not become a generic dumping ground or hide feature-specific business
behavior.

Shared UI primitives own reusable visual structure, accessibility mechanics,
and generic interaction behavior. Features compose them and retain workflow and
domain decisions. Feature modules must not recreate semantic tokens, base form
controls, dialogs, feedback patterns, or equivalent primitives as a competing
design system.

The detailed visual and interaction contract is owned by the
[frontend design system](../frontend-design-system.md).

## Design-token ownership

The application has one semantic token source. Presentation code consumes
purpose-based tokens rather than raw reusable visual values. Feature-specific
visual needs should first be expressed through existing semantics; new tokens or
variants require evidence that the distinction is stable and reusable.

Token values may be customized per project without changing component contracts.
Token naming and review rules are defined by the design system.

## Server state and client state

Server state includes remote data, loading, error, freshness, pagination, and
cache invalidation. Client state includes ephemeral UI state such as an open
dialog, a draft, focus, or local selection. Keep them distinct.

- Give remote requests stable identities that include every effective query
  input.
- Deduplicate equivalent in-flight work when useful and define cancellation
  semantics.
- After mutation, update or invalidate every affected projection and prevent
  stale in-flight responses from restoring old data.
- Preserve confirmed visible data during safe background refreshes.
- Do not cache without an understood invalidation strategy.
- Do not use optimistic updates unless rollback and reconciliation are defined.

## API clients and mapping

API clients belong to infrastructure. They own endpoint construction, request
serialization, response parsing, protocol error normalization, and DTO-to-domain
mapping. UI components must receive domain/application models and safe,
user-facing errors rather than raw protocol payloads or backend error messages.

## Lists and forms

Production lists should use backend pagination or another bounded loading
strategy. Search and filter parameters must be part of the remote query contract
when the complete relevant dataset is not loaded. Changing search or filters
normally resets pagination, and mutation refresh must preserve still-valid query
state.

Forms own interaction and draft state; domain/application code owns business
validation and normalization. Revalidate on submission, prevent duplicate
submission, preserve drafts on recoverable failure, and map field errors near
their controls.

## Permitted and prohibited imports

Permitted:

- presentation imports application, domain types, and shared UI primitives;
- application imports domain and its own ports;
- infrastructure imports application/domain contracts and external libraries;
- the composition root imports concrete feature adapters and entry points.

Prohibited:

- domain imports presentation, infrastructure, framework, or browser code;
- application imports components, styling, browser-only behavior, or concrete
  HTTP/storage implementations;
- presentation reaches around a use case into a concrete adapter;
- one feature imports another feature's internals to coordinate global state;
- features define independent global tokens or duplicate application chrome.

Cross-feature coordination belongs at composition boundaries or in an explicitly
shared contract justified by real use.

When a feature needs reference data owned by another feature, the consumer
defines the smallest purpose-specific application port and model it needs. The
composition root may satisfy that port with an existing adapter when the
contracts are structurally compatible. A consumer must not import another
feature's presentation code or richer domain model merely to populate a
selector or summary.

## Testing boundaries

- Domain tests cover pure business behavior.
- Application tests cover workflows and port interactions.
- Infrastructure tests cover request contracts, DTO mapping, errors, cache
  behavior, and adapter integration.
- Presentation tests cover observable loading, success, empty/no-result, error,
  retry, validation, duplicate-action prevention, keyboard, focus, and
  materially different responsive behavior.
- Application-shell tests cover routing, navigation, global providers, and
  cross-feature composition.

Avoid testing internal hook state when visible behavior can be tested. Snapshots
must not be the only evidence for important interactions.
