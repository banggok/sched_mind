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

The current foundation contains only an HTTP health endpoint. Scheduling
concepts and persistence will be introduced after their business requirements
and invariants are defined.

## Frontend

The frontend is a React and TypeScript application built with Vite. Tailwind CSS
provides utility-first styling through its official Vite plugin. Feature
boundaries should be introduced when actual user workflows are defined, rather
than creating speculative abstractions now.

Application bootstrap and page composition live in `src/app`. The current
`system-health` feature separates its HTTP adapter from its React presentation
state. The app composition root supplies the HTTP implementation to the
presentation hook as a plain function.

The health feature has no `domain` or `application` directory because it is a
technical connectivity indicator without business rules or use case
orchestration. Those layers should be introduced only when real responsibilities
require them.
