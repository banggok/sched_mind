# New-Project Documentation Bootstrap

This reusable workflow tells a coding agent how to establish project-specific
documentation after the engineering baseline is copied into a new repository.
It must run before feature implementation when `docs/project/` is absent or
incomplete.

## Outcome

Create non-empty, internally consistent documents:

```text
docs/project/
├── README.md
├── architecture.md
└── <project-slug>-context.md
```

The generated documents must describe the new repository. Never copy the
project-specific documents of the repository from which the baseline came.

## Agent workflow

### 1. Inspect before asking

Read the reusable baseline completely, then inspect the current workspace
without modifying it:

- existing README, product documents, user stories, ADRs, and diagrams;
- source and test directories;
- dependency manifests and lockfiles;
- build, lint, test, migration, container, CI, and run configuration;
- environment-variable examples without exposing secrets;
- current version-control status.

Classify each finding as confirmed repository fact, documented product decision,
or unresolved choice. Do not ask the user for information already established
reliably in the workspace.

### 2. Ask only for missing decisions

Ask concise questions in small related groups. Explain why each answer matters
and offer a recommendation when evidence supports one. At minimum, resolve the
applicable topics below.

Product context:

- product name, short purpose, and intended users or actors;
- business problem and expected outcome;
- initial scope and explicitly deferred scope;
- core terminology, entities, workflows, and known invariants;
- authoritative source for product requirements.

System shape:

- backend-only, frontend-only, full stack, service, library, CLI, worker, or
  another topology;
- repository organization and deployable units;
- required integrations and trust boundaries;
- synchronous, asynchronous, batch, or scheduled processing needs.

Technical choices not proven by the repository:

- language, runtime, framework, and supported versions;
- persistence, cache, queue, storage, and migration strategy;
- frontend framework, state approach, routing, styling, and design-token source;
- for projects with a UI, brand/theme inputs, accessibility target, supported
  viewport range, and the semantic token implementation that will apply the
  reusable design-system baseline by default;
- API or message contracts and compatibility expectations;
- local, test, staging, and production environment differences;
- authentication, authorization, privacy, and security constraints;
- observability, availability, performance, and scalability requirements;
- supported browsers, devices, databases, and operating environments.

Delivery and validation:

- local setup and run commands;
- formatting, lint, static analysis, test, build, migration, and smoke commands;
- test database isolation and external-service test strategy;
- CI/CD, deployment, release, and rollback expectations;
- documentation that must remain authoritative.

Do not force the user to decide a technology that is irrelevant to the product
or already fixed by repository evidence. Record unresolved choices explicitly
instead of inventing them.

### 3. Propose before writing

Summarize:

- confirmed facts;
- user decisions;
- assumptions that still require confirmation;
- proposed repository and architecture shape;
- files to create or update;
- conflicts with the reusable baseline, if any.

Wait for user approval before creating project-specific documents. Bootstrap
approval authorizes documentation creation, not application implementation.

### 4. Generate project documentation

`docs/project/README.md` must:

- declare the directory project-specific and unsafe to copy unchanged;
- link the reusable architecture and design-system baseline;
- link the generated architecture and context documents;
- explain explicit project exceptions and precedence;
- index authoritative product and engineering documents.

`docs/project/architecture.md` must contain only applicable, confirmed details:

- system context and deployable units;
- technology stack and supported versions;
- actual source and test layout;
- dependency and composition decisions;
- persistence, integrations, configuration, and process lifecycle;
- API, messaging, server/client state, and design-system implementation choices;
- the authoritative semantic token source and any confirmed project-specific
  design-system exceptions; feature requirements do not need to opt into the
  reusable design-system baseline;
- security, observability, performance, and operational constraints;
- local setup, validation, migration, run, and deployment commands;
- deliberate exceptions from reusable recommendations;
- unresolved technical decisions clearly marked as unresolved.

`docs/project/<project-slug>-context.md` must:

- identify the product and purpose;
- define actors and shared terminology;
- summarize current scope, deferred scope, workflows, and invariants;
- identify authoritative requirements and their precedence;
- distinguish confirmed rules from open product questions;
- avoid duplicating full acceptance criteria, PRDs, or user stories.

Derive `<project-slug>` from the confirmed product name using lowercase words
separated by hyphens. If the product name is unresolved, ask before choosing the
filename.

### 5. Validate and hand off

Before completion:

- verify every relative link;
- check that no template placeholders remain;
- confirm every technology and product claim is supported by repository evidence
  or an explicit user answer;
- check that reusable baseline files remain product-neutral;
- check that normative rules have one authoritative owner;
- list unresolved decisions without treating them as implemented constraints;
- show the final documentation tree and request confirmation.

After approval, coding may begin under `AGENTS.md` and the generated project
documents.

## Update behavior for an existing project

If `docs/project/` exists but is incomplete or stale, do not overwrite it from a
template. Inspect implementation and authoritative documents, identify exact
gaps or contradictions, ask only for unresolved decisions, and propose a scoped
update. Preserve valid project history and explicit exceptions.
