# Reusable Frontend Design System

This document owns reusable visual and interaction standards. It defines
contracts and decision criteria, not a product's brand palette or exact theme
values.

Normative terms:

- **Must** and **must not** are mandatory reusable standards.
- **Should** is a recommended convention that may be changed with a documented
  product or platform reason.
- **Customizable** values are expected to vary by project while retaining the
  semantic contract.
- A **project exception** must be explicit in project documentation and must not
  silently weaken accessibility or architectural invariants.

## Semantic token architecture

The frontend must have one authoritative token source. Components consume
semantic tokens; raw reusable values do not belong in feature code.

This design-system baseline applies automatically to every project with a user
interface. Product requirements do not need to request it explicitly. A new
frontend must establish its semantic token source and document its chosen theme
before feature presentation code introduces reusable visual decisions. This
does not authorize an empty component library: shared primitives remain
demand-driven as described below.

Use token layers where the platform supports them:

1. Foundation values describe raw scales such as neutral colors or spacing.
2. Semantic tokens describe purpose such as `text-primary`, `surface-canvas`,
   `border-strong`, `action-primary`, or `feedback-danger`.
3. Component tokens are permitted only when a stable component distinction
   cannot be expressed through semantic tokens.

Name tokens by purpose, not a feature, screen, literal color, or temporary
implementation. Token values are customizable; semantic meaning is mandatory.
The authoritative source must cover reusable color, typography, spacing,
sizing, layout, radius, border, elevation, motion, and layering decisions.

Semantic component and layout classes are a valid token-consumption boundary.
Feature code may use framework scale utilities for local composition when they
resolve to the shared scale, but it must not repeat a visual contract that
belongs to a shared primitive.

## Color semantics

The token set should distinguish:

- brand/action emphasis;
- primary, secondary, muted, disabled, and inverse text;
- canvas, surface, elevated surface, and selected surface;
- subtle and strong borders;
- focus indication;
- success, warning, danger, and informational feedback;
- overlays and status indicators.

Interactive states must remain distinguishable without relying on color alone.
Text and meaningful controls must meet the project's adopted accessibility
contrast target. Exact brand colors are project-customizable.

## Typography

Typography tokens should describe role: display, page title, section heading,
body, label, caption, and code. Each role should define a coherent font family,
size, line height, weight, and optional tracking.

Projects may customize families and scales. Components must not invent isolated
font sizes or weights when an existing role communicates the same hierarchy.
Text must tolerate reasonable browser zoom and content growth.

## Spacing and layout

Use a shared spacing scale for gaps, padding, and rhythm. Prefer layout
composition over feature-specific magic numbers. Customizable breakpoints,
content widths, and density may reflect the target device and data volume.

Page composition should provide:

- stable application chrome;
- a predictable content container and reading order;
- consistent page header, primary action, filters, and content regions;
- reserved scrollbar space or an equivalent strategy when layout shift would
  move global chrome;
- no horizontal scrolling for ordinary page content.

## Radius, borders, and elevation

Use semantic radius roles such as action, control, panel, and elevated surface.
Use border roles for hierarchy and state rather than choosing arbitrary colors
per component. Use the smallest elevation that communicates stacking.

Projects may customize exact values, but equivalent components must use the same
role. Shadows must not be the only indication of an interactive or selected
state.

## Motion

Motion should clarify state change, not delay work. Keep durations and easing on
a shared scale, avoid unnecessary movement, and respect reduced-motion
preferences. Do not animate synchronous feedback merely for decoration.

## Layering and z-index

Define a small named layering scale for content, sticky chrome, dropdowns,
overlays, dialogs, and transient notifications. A component must not win a
stacking conflict by adding an arbitrary larger z-index. Nested overlays require
an explicit stacking and focus contract.

## Shared primitive contracts

Shared primitives own consistent styling, generic states, keyboard behavior,
focus visibility, and ARIA mechanics. Their APIs must stay narrow and must not
hide business decisions.

Common candidates include buttons, fields, selectors, dialogs, popovers,
calendars, search inputs, pagination, feedback messages, and skeletons. Create a
primitive only after reuse or a stable application-wide need is demonstrated.
Mandatory application-wide accessibility behavior, such as dialog focus
containment, is sufficient evidence for a primitive even before multiple
features use it.

Equivalent controls must reuse an existing primitive. Add a variant only when
the semantic role or interaction materially differs; do not add variants solely
to reproduce one screen's incidental styling.

## Page composition and navigation

- Global chrome must remain structurally stable across page navigation.
- Current location and active navigation must be visible.
- Primary navigation should reset the viewport unless the workflow explicitly
  requires restoration.
- Keep the primary action easy to identify and avoid competing primary actions.
- Preserve unsaved work or warn before discarding it.
- Do not duplicate shell elements inside feature pages.

## Forms

- Use visible labels as the primary field description and identify required
  fields.
- Associate labels, help text, and errors with controls.
- Keep the accessible label limited to the field name. Required indicators must
  not alter that name, and help or error text must be associated through a
  description relationship rather than included inside the label.
- Preserve input after validation or submission failure.
- Validate as early as useful without interrupting normal entry; always validate
  again on submit.
- Put actionable field errors near the affected control and focus the first
  invalid field when useful.
- Prevent duplicate submission and disable only affected controls.
- Do not silently transform input unless the transformation is obvious and safe.
- Confirm destructive or irreversible actions with specific language.

Exact field density and layout are customizable. Accessibility and failure
recovery are mandatory.

## Actions

Use a clear hierarchy such as primary, secondary, quiet, and destructive.
Primary actions represent the main safe progression. Destructive actions must
not be the visually dominant default. Icon-only actions require meaningful
accessible names and adequate touch targets.

Every asynchronous action must expose appropriate pending, success, and failure
feedback. Do not announce success before confirmation or leave users uncertain
whether an action completed.

## Tables and lists

- Clearly communicate sorting, search/filter state, pagination, selection, and
  result counts when available.
- Distinguish an empty dataset from no matching results and load failure.
- Production datasets must use a bounded strategy such as backend pagination,
  virtualization, or incremental loading.
- Remote search should be debounced only when it reduces waste without hiding
  intent; a search/filter change normally resets to the first page.
- Prefer meaningful labels over raw identifiers.
- Provide a responsive representation when a full table cannot remain usable.

Whether search is applicable and the default page size are product or project
decisions, not universal design-system constants.

## Loading, empty, error, and success states

- Initial backend-backed sections must show a local loader or shape-preserving
  skeleton when waiting is perceptible.
- Preserve confirmed content during safe background refreshes.
- Empty, no-result, unavailable, permission-denied, and load-failure states must
  be distinguishable.
- Empty and no-result states should provide the next useful action when one
  exists.
- Errors must use understandable language, avoid internal details, preserve
  recoverable work, and provide Retry or another recovery action when possible.
- Success feedback must reflect confirmed state. Toasts may supplement but must
  not be the sole location of critical information.

## Accessibility baseline

- Prefer semantic HTML and native behavior before ARIA.
- Support keyboard operation and visible focus for every interactive element.
- Provide accessible names for controls and announce dynamic status or errors.
- Do not communicate meaning through color alone.
- Maintain sufficient contrast, reasonable text resizing, and reduced-motion
  support.
- Dialogs and popovers must define focus entry, containment where appropriate,
  Escape behavior, close behavior, and focus return.
- Responsive behavior must be tested on supported sizes; do not assume hover or
  precise pointer input.
- Add automated accessibility checks for stable shared primitives and critical
  workflows when the frontend test platform supports them. Automated checks
  complement rather than replace keyboard, focus, contrast, zoom, screen-reader,
  and responsive review.

## Adding tokens or variants

Before adding a token:

1. Confirm no existing semantic role fits.
2. Demonstrate reuse or a stable system-level distinction.
3. Name the purpose without product or feature terminology.
4. Define required states, contrast, and theming behavior.
5. Update this document only if the reusable contract changes.

Before adding a component variant:

1. Confirm the difference is semantic or behavioral, not incidental.
2. Reuse existing tokens and interaction mechanics.
3. Add behavior and accessibility tests where materially different.
4. Document project-specific exceptions outside this reusable baseline.

## Prohibited patterns

- Raw reusable colors, shadows, radii, or typography values in feature code.
- Feature-specific global token sets or duplicate base primitives.
- Token names tied to a feature, screen, or literal color.
- Arbitrary z-index escalation.
- Placeholder-only labels or color-only status.
- Full-page blocking for a local operation.
- Unrecoverable form failure that discards valid input.
- New interaction patterns when an established primitive already satisfies the
  requirement.

Arbitrary literal values are permitted only for intrinsic asset geometry,
viewport calculations, or a documented third-party constraint that cannot be
expressed by the shared scale. Keep such values at the application or shared
primitive boundary. If the same exception recurs, promote it to a semantic
token; feature code must not accumulate one-off arbitrary visual values.

## Review checklist

- Does every visual value use the correct semantic role?
- Is hierarchy consistent across equivalent pages and controls?
- Are pending, empty, no-result, error, and success states distinct?
- Can users recover from failures without losing valid work?
- Are destructive actions protected and visually subordinate?
- Are keyboard, focus, labels, announcements, contrast, resizing, and reduced
  motion handled?
- Does the layout remain usable on supported sizes and input methods?
- Does a new token or variant represent a reusable distinction?
- Are project-specific brand values and exceptions documented outside this file?
