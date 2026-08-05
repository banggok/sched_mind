# SchedMind

> **Capacity-aware planning for predictable engineering delivery.**

SchedMind helps engineering teams turn project scope into a realistic execution plan based on actual team capacity, task dependencies, project priorities, working calendars, and delivery buffers.

Most project planning tools allow teams to enter dates manually. Those dates often become outdated as soon as priorities, assignments, capacity, or dependencies change.

SchedMind approaches planning differently. It calculates and continuously updates the schedule from the constraints that determine when work can actually be completed.

---

## Why SchedMind?

Engineering plans commonly fail for predictable reasons:

- Task dates are estimated independently from engineer capacity.
- The same person is assigned to overlapping work across multiple projects.
- Dependencies are documented but not reflected consistently in the schedule.
- Public holidays, leave, support duties, and reduced availability are handled manually.
- Project plans become stale after assignments or priorities change.
- Teams commit to delivery dates without a visible planning buffer.
- Sprint plans are disconnected from the project schedule.

SchedMind brings these planning inputs into one system so that schedule changes can be calculated consistently instead of corrected manually across multiple spreadsheets.

---

## Who Is It For?

SchedMind is designed primarily for:

- Engineering Leads
- Engineering Managers
- Technical Delivery Leads
- Project and Program Leads
- Teams managing shared engineers across multiple projects

It is especially useful when engineers work across several initiatives and delivery dates depend on limited or changing capacity.

---

## What SchedMind Helps You Do

### Build realistic project plans

Break projects down into a hierarchical Work Breakdown Structure and define the information required to schedule executable work:

- Task hierarchy
- Role
- Assignee
- Effort
- Capacity allocation
- Dependency
- Scheduling lag
- Execution and commitment dates
- Actual dates

Groups and projects automatically summarize the schedule and completed effort of all tasks beneath them.

### Schedule work using actual capacity

SchedMind calculates task schedules using the capacity available to each team member.

The calculation can account for:

- Daily working capacity
- Member planning buffer
- Project delivery buffer
- Partial allocation to a task
- Public holidays
- Temporary capacity overrides
- Existing work across multiple projects
- Project priority
- Task order
- Dependencies between tasks

This reduces the risk of creating a plan that assumes the same engineer can perform several full-time tasks simultaneously.

### Separate the working plan from the delivery commitment

SchedMind provides two planning perspectives:

- **Execution** represents the working schedule based on operational capacity.
- **Commitment** adds the configured project buffer to provide a safer delivery commitment.

This allows teams to plan execution aggressively while communicating a more controlled commitment externally.

### Automatically recalculate affected work

When automatic scheduling is enabled, SchedMind recalculates the relevant unfinished work after scheduling inputs change.

Examples include:

- A task receives a different assignee.
- Effort changes.
- A dependency is added or removed.
- Team capacity changes.
- A public holiday is added.
- Project priority changes.
- A project is reopened.

Before changes that may affect other projects are applied, SchedMind can identify the impacted projects and require confirmation.

### Manage dependencies clearly

Tasks can block other tasks through explicit dependencies.

SchedMind uses those relationships when determining when work can begin. Dependency arrows are also shown in the portfolio Gantt so scheduling constraints remain visible during planning.

### Compare multiple projects in one workspace

The Home workspace provides a portfolio-level Gantt view across active projects.

From one screen, users can:

- Review multiple projects and their WBS hierarchy.
- Switch between Execution and Commitment schedules.
- Filter the portfolio by project and role.
- Save useful project-filter combinations.
- Review task bars and dependency relationships.
- Open project, group, and task actions without leaving the workspace.
- Reorder or move WBS items through the established planning workflow.

This helps expose conflicts that are difficult to see when every project is planned separately.

### Recommend an assignee based on schedule impact

SchedMind can evaluate eligible team members for an unfinished task and estimate the scheduling result for each candidate.

Recommendations consider:

- Role eligibility
- Existing allocations
- Current project priority
- Task effort
- Capacity allocation
- Scheduling mode
- Estimated completion
- Remaining capacity on the completion date
- Additional overcapacity introduced

The recommendation supports the decision. The user still selects the final assignee.

### Plan sprints without creating a second schedule

Sprint Planning groups selected members and scheduled tasks into an execution-facing daily plan.

SchedMind can:

- Suggest tasks from the current Execution schedule.
- Include work that should finish within the sprint period.
- Show daily member capacity.
- Show task allocation for each sprint date.
- Surface overcapacity without changing the project schedule.
- Allow tasks to be reviewed and selected manually.
- Preserve the authoritative project schedule as the source of truth.

A Sprint organizes scheduled work. It does not move tasks, reserve capacity, or create a separate scheduling engine.

### Protect approved plans

Projects have a lifecycle:

- **Open** — planning and scheduling changes are allowed.
- **Locked** — the approved Execution and Commitment plan is protected.
- **Closed** — the project is completed and removed from active planning views.

A Locked project must be reopened before its planning data can be changed. This prevents routine scheduling changes from silently altering an approved baseline.

### Record actual completion without losing planning history

Actual Start and Actual End can be recorded for executable tasks.

Completed tasks remain historical anchors. When a completion entry is incorrect, the task can be reopened without recreating it or losing its hierarchy, dependency, and planning information.

---

## Typical Workflow

### 1. Configure the team

Create the roles used by the team, register team members, and define their daily capacity and planning buffer.

Add temporary capacity overrides when availability changes because of leave, support work, training, or other operational commitments.

### 2. Configure the working calendar

Register public holidays so the scheduler does not allocate work on unavailable dates.

### 3. Create a project

Create a project and configure:

- Automatic or manual scheduling
- Scheduling start date
- Project buffer
- Project priority

### 4. Break down the work

Create groups and executable tasks in the project WBS.

Assign roles, engineers, effort, allocation percentage, lag, and dependencies.

### 5. Review the generated schedule

Use the Home portfolio Gantt to review:

- Cross-project allocation
- Task sequence
- Dependencies
- Execution dates
- Commitment dates
- Project and group summaries

Adjust priority, assignment, effort, allocation, or dependency when the generated result does not match the intended delivery strategy.

### 6. Lock the approved plan

Once the plan is ready, lock the project to protect its Execution and Commitment baseline.

### 7. Prepare sprint execution

Create a Sprint, select participating members, and review the suggested tasks and daily allocation plan.

### 8. Track actual completion

Record actual dates as tasks are completed. Reopen a task when completion was recorded incorrectly or the work must continue.

---

## Core Capabilities

| Area | Capability |
| --- | --- |
| Team configuration | Roles, team members, daily capacity, and member buffer |
| Availability | Public holidays and temporary capacity overrides |
| Project planning | Project lifecycle, priority, automatic/manual scheduling, and project buffer |
| Work breakdown | Unlimited WBS hierarchy with Group and Task presentation |
| Task planning | Role, assignee, effort, allocation, lag, dependencies, manual dates, and actual dates |
| Scheduling | Capacity-aware Execution and Commitment scheduling |
| Portfolio view | Multi-project Gantt, filters, saved views, summaries, and dependency visualization |
| Assignment support | Schedule-based assignee recommendation |
| Sprint planning | Member selection, task suggestion, daily capacity, and task allocation |
| Plan protection | Project locking, closing, reopening, and impact confirmation |
| Progress handling | Actual completion and task reopening |

---

## Local Usage

SchedMind currently runs locally on each user's computer.

Each local installation has its own application instance and PostgreSQL database. Data is not automatically shared between users or synchronized with another installation.

### Prerequisites

Install these applications once:

- Git
- Go 1.24 or later
- Node.js 20.19 or later
- npm 10 or later
- Docker Desktop

The commands below are intended for Terminal on macOS or Linux. Windows users need a compatible shell such as WSL or Git Bash.

Make sure Docker Desktop is running before installing or starting SchedMind.

### Clone the repository

```sh
git clone https://github.com/banggok/sched_mind.git
cd sched_mind
```

### Install SchedMind

Run this command after cloning the repository:

```sh
./scripts/install.sh
```

The installer:

- verifies the required applications and versions;
- creates the local environment configuration when needed;
- downloads backend dependencies;
- installs frontend dependencies;
- starts the local PostgreSQL database; and
- waits until the database is ready.

The installer is safe to run again after receiving an application update. It preserves the existing environment configuration and local database.

### Start SchedMind

```sh
./scripts/run-all.sh
```

Open the following address in a browser:

```text
http://localhost:5173
```

Press `Ctrl+C` in Terminal to stop the backend and frontend.

The PostgreSQL container remains running so the next application startup is faster.

For normal daily use, only this command is required:

```sh
./scripts/run-all.sh
```

### Update SchedMind

```sh
git pull --ff-only
./scripts/install.sh
./scripts/run-all.sh
```

Running the installer after an update ensures that changed backend or frontend dependencies are installed.

---

## Current Deployment Model

The current version is intended for local evaluation and individual usage.

Important characteristics:

- Each installation stores its own local data.
- There is no shared central database.
- Changes made by one user are not visible to another user.
- Authentication and organization-level permissions are not included.
- Data backup and transfer between installations are not yet managed through the product interface.
- Concurrent multi-user editing is not supported by the local deployment model.

These constraints should be considered before using SchedMind as the authoritative planning system for an entire organization.

---

## Data Storage

SchedMind uses PostgreSQL 16 running in Docker.

Application data is stored in a persistent Docker volume. Stopping the application does not delete the database.

To stop the local database without deleting its data:

```sh
docker compose stop postgres
```

Do not run commands that delete Docker volumes unless the local SchedMind data is no longer required.

---

## Development

Developers can run the backend and frontend separately after completing installation.

### Backend

```sh
./scripts/run-backend.sh
```

The backend API is available at:

```text
http://localhost:8080
```

Health endpoints:

```text
GET /health
GET /api/health
```

### Frontend

```sh
./scripts/run-frontend.sh
```

### Validation

Run all backend and frontend validation steps:

```sh
./scripts/validate.sh
```

The validation script stops immediately when a step fails.

A successful run ends with:

```text
All validation steps passed.
```

---

## Product Direction

SchedMind is focused on one core question:

> Given the work, dependencies, priorities, and capacity available today, when can the team realistically deliver?

The product is designed to make that answer visible, explainable, and easier to maintain as planning conditions change.
