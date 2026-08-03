# US-6.4 Remove Auto Dependency Implementation Evidence

| AC group | Production evidence | Automated evidence |
| --- | --- | --- |
| AC-1–17, 27–28 | Manual-only dependency domain/repository/HTTP; single-pass scheduler; date-only preview; manual-only frontend gateways and UI | Dependency, scheduling, WBS preview, portfolio, and rendered frontend suites |
| AC-18–26 | Migration 000023 and `MigrateWithPostStep` atomically clean schema/data and recalculate the active portfolio | Migration SQL inspection and backend scheduler suites; PostgreSQL startup smoke remains environment-dependent |
| AC-29–30 | Updated stories, architecture, context, and this traceability record | Repository validation commands and actual results in the completion report |

The maintenance query scans/deletes legacy `task_dependencies` once. Existing
endpoint uniqueness, foreign keys, and directional indexes remain unchanged.
Normal scheduling no longer issues ownership queries or dependency DML.
