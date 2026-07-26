# Coding Agent Instructions

These instructions are mandatory for every coding agent working in this
repository. Detailed standards have a single authoritative owner and must be
read through the links below rather than duplicated here.

## Required reading

Before implementation, read:

1. [Architecture entry point](docs/architecture.md).
2. The applicable reusable [backend](docs/architecture/backend.md) or
   [frontend](docs/architecture/frontend.md) architecture.
3. The [frontend design system](docs/frontend-design-system.md) for UI work.
4. [Project documentation](docs/project/README.md), including its architecture,
   product context, and authoritative requirement links.

Read referenced documents completely. Do not infer a business rule from code or
generic guidance when an authoritative product document exists.

If `docs/project/README.md`, `docs/project/architecture.md`, or a project-context
document does not exist, treat the repository as not yet initialized. Before
coding, follow the [project bootstrap](docs/project-bootstrap.md), inspect the
workspace, ask the user for missing decisions, and generate the project-specific
documents. Do not guess answers or copy another product's project documents.

## Architecture and implementation

- Preserve dependency direction toward the business core.
- Keep business rules in the domain and workflows in the application layer.
- Keep transport mapping, persistence, framework code, and integrations outside
  the domain.
- Keep frontend business features within their feature boundaries.
- Use existing shared UI primitives and semantic design tokens; feature modules
  must not create a competing design system.
- Separate entities, API DTOs, and persistence models when their responsibilities
  differ.
- Define interfaces only at meaningful substitution or architectural boundaries.
- Do not introduce speculative layers, abstractions, extension points, or empty
  architecture folders.
- Preserve observable behavior during refactoring and avoid unrelated changes.
- Do not add dependencies or change public contracts without approval.
- Never manually edit generated code.
- Before adding a local technical helper, search the repository for an existing
  equivalent. Promote confirmed identical, context-free technical behavior to
  a narrowly named shared package instead of duplicating it across features.
  Do not merge business rules, domain validation, DTO mapping, or error mapping
  merely because their current implementations look similar; bounded-context
  meaning takes precedence over superficial deduplication.
- TypeScript source and tests owned by the project must not use explicit or
  implicit `any`. Define the actual type when it is known. At untrusted or
  dynamically typed boundaries, accept `unknown` and narrow or validate it
  before use. Do not bypass this rule with `as any`, lint suppression, or an
  unnecessarily broad generic. Generated code and third-party declarations are
  outside this rule and must not be manually rewritten merely to remove their
  use of `any`.

## Business requirements

- Treat approved user stories and explicitly identified product documents as the
  source of truth for product behavior.
- Do not invent, weaken, or silently reinterpret business rules or acceptance
  criteria.
- Stop and explain before implementation if a requirement contradicts another
  authoritative requirement or a mandatory architectural invariant.
- Keep terminology consistent across domain code, APIs, UI, tests, and
  documentation while respecting each boundary's responsibility.

## Integrations, persistence, and operations

- Obtain environment-specific integration configuration from environment
  variables; never hardcode credentials, endpoints, or environment identifiers.
- Keep domain and application code independent of database and transport
  frameworks.
- Design production queries with their indexing, ordering, and supported-dialect
  behavior; follow the project-specific persistence constraints.
- For every new or changed production query, complete the query-review gate in
  the reusable backend architecture before completion. Do not wait for a user
  story to request query optimization explicitly.
- Long-running services must shut down gracefully with a bounded timeout and
  release owned resources.
- Preserve error causes internally, map errors at boundaries, and never expose
  infrastructure details to users.
- When a returned Go object is invalid on error, return a pointer and `nil, err`.
  Check a returned pointer for nil before dereferencing it, even when the error
  is nil; treat unexpected `nil, nil` as a contract violation.

## UX and design system

- For every project with a user interface, apply the reusable frontend design
  system by default even when the user story or PRD does not mention design
  tokens, shared primitives, UX consistency, or accessibility. These are
  baseline completion requirements, not optional features.
- During a new frontend project's bootstrap, establish one semantic token
  source and record its project-specific theme decisions before implementing
  feature UI. Create shared primitives only when the first real consumer or a
  mandatory application-wide accessibility contract justifies them; do not
  scaffold speculative component libraries.
- Treat usability, accessibility, feedback, consistency, and recovery as product
  requirements.
- Every asynchronous user action and backend-backed section must expose an
  appropriate loading, success, empty/no-result, or recoverable error state.
- Prevent duplicate submissions, preserve valid user input on failure, and
  confirm destructive actions when they are not easily reversible.
- Use semantic HTML, keyboard-accessible controls, visible focus, associated
  labels, sufficient contrast, and responsive layouts.
- Use the shared semantic token source and established primitives. Do not place
  raw reusable visual values in feature components.
- During frontend planning and self-review, explicitly assess token coverage,
  primitive reuse, responsive behavior, interaction states, and accessibility;
  requirements do not need to repeat these baseline obligations.

## Documentation ownership

- For every behavior, API, database, configuration, architecture, or UX change,
  assess its documentation impact.
- Update `AGENTS.md` only when mandatory agent behavior changes.
- Update reusable architecture or design-system documents only for deliberately
  reusable standards, never for an accidental current implementation.
- Put repository-specific technical decisions under `docs/project/`.
- Update the affected user story when observable product behavior, acceptance
  criteria, terminology, or mandatory tests change.
- Do not duplicate normative rules. Link to their authoritative owner.

## Required workflow

1. Confirm project documentation exists; otherwise complete the documented
   project-bootstrap workflow and obtain user confirmation.
2. Understand the requirement, relevant code, boundaries, documentation, and
   existing worktree changes.
3. For non-trivial work, state the approach, affected files, assumptions, and
   risks before editing.
4. Implement only the approved scope with minimal, cohesive changes.
5. Add behavior-focused tests covering applicable success, validation, business
   failure, dependency failure, boundaries, and concurrency.
6. Run all relevant formatting, static analysis, test, build, migration, and
   smoke-test commands documented in project architecture.
7. Restart a changed backend before local verification and confirm the new
   process and an affected endpoint.
8. Self-review architecture, security, performance, accessibility, edge cases,
   test quality, documentation consistency, and unnecessary abstractions.

## Completion report

Report the implemented outcome, root cause for bug fixes, behavior affected,
files and migrations changed, architectural decisions, tests and validation
results, documentation impact, deviations, assumptions, remaining risks, and
technical debt. Do not claim completion while mandatory validation is failing.
