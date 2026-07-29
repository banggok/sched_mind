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

## Acceptance-Criterion Confidence Standard

An Acceptance Criterion is not considered implemented until it is supported by all three confidence levels below.

### 1\. Code Inspection

The implementation must be inspected against the Acceptance Criterion and all relevant business invariants.

The inspection must confirm:

- the complete success path is implemented;
- all rejection and error paths required by the AC are implemented;
- business rules are enforced at the correct architectural boundary;
- frontend restrictions are not used as the only enforcement of a business invariant;
- persistence, transaction, concurrency, and rollback behaviour are considered where relevant;
- no adjacent behaviour contradicts the AC;
- the implementation does not rely only on incidental framework behaviour.

Code inspection alone is not sufficient evidence that an AC is complete.

### 2\. Unit or Integration Test

Each AC must have automated tests at the lowest meaningful boundary that can prove its business behaviour.

Use:

- domain or application unit tests for isolated business rules;
- repository integration tests for persistence, constraints, transactions, locking, rollback, and relation integrity;
- component tests for local frontend interaction and state behaviour;
- infrastructure tests for cache, request coordination, mapping, and external adapter behaviour.

Tests must verify observable outcomes, not only execution coverage.

Where relevant, tests must include:

- the successful path;
- rejection or validation paths;
- preserved state after failure;
- persistence state after success or failure;
- identity and relation integrity;
- concurrency, duplicate submission, or stale-response behaviour;
- transaction rollback.

A passing test suite is not proof that an AC is covered unless a test can be explicitly traced to that AC.

### 3\. Acceptance-Level Test

Each AC must have automated evidence through the highest practical public boundary of the behaviour.

Examples:

- HTTP API tests that execute the request through routing, transport mapping, application logic, and persistence;
- frontend feature tests that exercise the user-visible workflow through the rendered component and gateway contract;
- end-to-end tests when the behaviour depends on integration between independently deployed components;
- contract tests when an external or cross-module API contract is the acceptance boundary.

An acceptance-level test must verify behaviour observable by the actor or API consumer, including:

- the action performed;
- the resulting visible or returned state;
- the relevant error contract;
- the absence of forbidden side effects.

Tests that call an internal helper, repository method, React hook, or component implementation detail do not qualify as acceptance-level evidence.

### AC Completion Rule

For every Acceptance Criterion, implementation work must produce a traceability record containing:

Confidence levelRequired evidenceCode inspectionRelevant production files and inspected invariantsUnit/integrationTest file and test case namesAcceptance-levelTest file and user/API workflow covered

An AC may be marked Implemented only when all three rows contain valid evidence.

Use these statuses:

- Implemented: all three confidence levels are satisfied;
- Partially implemented: production behaviour exists, but one or more confidence levels are missing;
- Not implemented: required behaviour is absent or incorrect;
- Blocked: verification cannot be completed because of an external constraint.

Do not report an AC as implemented merely because:

- the production code appears correct;
- existing tests pass;
- coverage percentage is high;
- a neighbouring AC has similar tests;
- the frontend hides an invalid operation;
- the database happens to reject invalid data without an explicit contract;
- the behaviour was manually tested but has no required automated evidence.

### Audit and Implementation Workflow

When implementing or auditing a user story:

1.  Read the Acceptance Criteria in order.
2.  Evaluate each AC independently against all three confidence levels.
3.  Stop at the first AC that is not fully satisfied when the task explicitly requires sequential auditing.
4.  Implement or repair only the identified gap unless broader changes are technically required.
5.  Add or update tests for every missing confidence level.
6.  Run the relevant formatting, static analysis, unit, integration, acceptance, race, and build checks.
7.  Report the exact evidence for each verified AC.
8.  Never claim that unexecuted tests passed. State explicitly when a test could not be executed and why.

### Test Quality Rules

Tests must:

- assert business outcomes and side effects;
- fail when the AC behaviour is removed or reversed;
- avoid assertions that only repeat mock configuration;
- use stable identifiers when identity preservation matters;
- inspect persisted state when database state is part of the AC;
- inspect both sides of bidirectional relations when relation projection is part of the AC;
- verify that forbidden replacement or implicit behaviour did not occur;
- remain deterministic and independent of execution order.

One test may provide evidence for multiple ACs, but every AC must explicitly reference the relevant assertions. Merely sharing a test file is not sufficient.
