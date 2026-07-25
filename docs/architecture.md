# Architecture

The project uses pragmatic Domain-Driven Design boundaries. Dependencies point
toward the business core.

## Backend layers

- `domain` owns business entities, value objects, invariants, services, and
  domain errors. It has no transport, persistence, framework, or infrastructure
  dependencies.
- `application` coordinates use cases and transaction boundaries. Repository
  interfaces belong here when they are consumed by a use case.
- `transport` validates and maps incoming requests and outgoing responses. It
  does not implement business rules.
- `infrastructure` implements persistence and external integrations. It does not
  implement business rules.

The backend currently exposes the technical health endpoint and the `roles` and
`teammembers` business features. Each business feature owns its domain,
application, transport, and infrastructure packages. PostgreSQL is used for the
local runtime, while isolated automated tests may use SQLite through the same
GORM repository boundary.

Top-level backend packages are organized by responsibility without pretending
technical concerns are business domains. `internal/httpapi` is the HTTP
composition point that registers feature transports. The technical
`internal/systemhealth` feature owns its HTTP health handler and intentionally
has no domain or application layer because it has no business rules or use-case
orchestration.

Production list use cases accept the shared `listing.Query` value and return a
`listing.Page`. HTTP list endpoints expose `search`, `page`, and `pageSize`
query parameters and return `page`, `pageSize`, and `total` metadata. The
default page size is 5 and the transport caps it at 100. Search and pagination
remain application concerns; persistence implementations apply the actual
filter, deterministic ordering, count, limit, and offset.

Current name searches use a case-insensitive prefix match so the database can
serve them with an index. PostgreSQL migrations define functional
`LOWER(name) text_pattern_ops` indexes. A future MySQL infrastructure migration
must provide the equivalent indexed access path, using an indexed generated
lowercase column or a compatible indexed collation according to the supported
MySQL version, without changing domain or application contracts.

## Frontend

The frontend is a React and TypeScript application built with Vite. Tailwind CSS
provides utility-first styling through its official Vite plugin. Business code
is organized by feature, and each feature introduces only the architectural
layers justified by its current behavior.

Application bootstrap, navigation metadata, and page composition live in
`src/app`. Global page chrome is split into `ApplicationSidebar`, `TopBar`,
`AppShell`, and `PageContent`, so feature pages do not own or duplicate global
layout behavior. `App` owns the single persistent `AppShell`; route changes
replace only its feature content, update its active navigation metadata, and
reset the document viewport to the top-left. Feature pages must not instantiate
the shell or implement their own primary-route scroll restoration.

`src/app/navigation.ts` is the single source of truth for page IDs, labels,
hashes, ordering, and navigation groups. The sidebar and breadcrumb derive their
visible metadata from this registry. Adding a page must not require duplicating
its menu or breadcrumb labels across feature components.

The `roles` and `team-members` features keep domain models and validation
separate from API DTOs. Their infrastructure adapters map HTTP payloads at the
feature boundary, and presentation code consumes application ports and use
cases.

Presentation components shared by multiple features, such as the standard list
search field, pagination controls, and list loading skeleton, live in
`src/shared/presentation`. Feature-specific result-state decisions remain
inside their feature presentation boundary. Production list searches are
debounced, execute against the backend, and reset pagination to page one;
feature pages never fetch an unbounded collection merely to search or paginate
it in the browser.

HTTP gateways use the shared request-cache primitive in
`src/shared/infrastructure` to deduplicate concurrent identical list loads and
reuse completed list results across page navigation. Individual consumers may
abort without cancelling shared work. Successful create, update, and delete
operations invalidate the related list cache. Versioned invalidation prevents
an older in-flight response from repopulating a cache after a mutation.
Because member projections embed the current role name, the application
composition root wires successful Role mutations to invalidate the Members list
cache as well. Features remain unaware of one another; `App` coordinates this
confirmed cross-feature cache dependency.

The health feature has no `domain` or `application` directory because it is a
technical connectivity indicator without business rules or use case
orchestration. Those layers should be introduced only when real responsibilities
require them.
