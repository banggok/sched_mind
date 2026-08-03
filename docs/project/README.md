# Project-Specific Documentation

Documents in this directory describe the current product and repository. They
are not part of the reusable engineering baseline and must be reviewed,
replaced, or removed when the baseline is copied to another project.

The reusable baseline consists of:

- [`AGENTS.md`](../../AGENTS.md)
- [`docs/architecture.md`](../architecture.md)
- [`docs/architecture/backend.md`](../architecture/backend.md)
- [`docs/architecture/frontend.md`](../architecture/frontend.md)
- [`docs/frontend-design-system.md`](../frontend-design-system.md)
- [`docs/project-bootstrap.md`](../project-bootstrap.md)

This directory supplies the concrete decisions needed to apply that baseline:

- [`architecture.md`](architecture.md) records the current technology stack,
  repository layout, persistence choices, runtime topology, and deliberate
  implementation decisions.
- [`schedmind-context.md`](schedmind-context.md) summarizes product and domain
  context and points to authoritative requirements.
- [`task-capacity-allocation-implementation-evidence.md`](task-capacity-allocation-implementation-evidence.md)
  records US-6.3 and revised US-6.2 acceptance traceability.

Project-specific rules may override a generic recommendation only when the
exception is explicit, justified, and documented here. Mandatory dependency,
security, data-integrity, or accessibility invariants must not be silently
overridden. A real conflict must be resolved deliberately and recorded rather
than implemented by assumption.
