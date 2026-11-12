# YAP (Yet Another Planner)

YAP is a personal and team project planner built around a local-first design. It works fully offline for a single user and can optionally connect to a server for team collaboration and multi-device synchronization.

## What It Is

YAP organizes work as projects, Kanban boards and tasks. A user can start using it immediately without creating an account. All data lives in a local SQLite database on the user's device. When a team needs to collaborate, the same data model is shared through a server backed by PostgreSQL.

## Core Idea

The same logical schema exists in two physical databases:

| | Local | Server |
| --- | --- | --- |
| Engine | SQLite | PostgreSQL |
| Scope | The user's own device | Team workspaces, accounts, invitations, synchronization |

- **Personal use:** Frontend, Backend and SQLite only. No account, no network.
- **Team use:** The server is the source of truth for team workspaces; the local database acts as a cache.
- **Personal workspace sync:** The server only stores a backup and synchronizes data across the user's devices.

## Domain Model

| Concept | Description |
| --- | --- |
| User | Identity. Exists even without an account, so IDs never need rewriting when an account is created later. |
| Account | Login credentials (email, password hash). Server-only, never synchronized. |
| Workspace | Authorization and synchronization boundary. Either `personal` (single member) or `team` (multiple members). |
| Membership | A user's role in a workspace: `owner`, `admin`, `member` or `viewer`. |
| Team | A flat, permanent group of people inside a workspace (for example Backend, Frontend). |
| Project | A unit of work belonging to a workspace, associated with one or more teams. |
| Status | A Kanban column of a project, categorized as `todo`, `in_progress` or `done`. |
| Task | A unit of work in a project and status. It can have one owning team, assignees and subtasks. |
| Comment | A note a member leaves on a task. Only its author can edit or delete it. |

## Design Decisions

- **Soft deletion everywhere.** Rows are marked with `deleted_at`, `removed_at` or `left_at` so that deletions can be synchronized. Physical deletion happens only in a purge job.
- **UUIDv7 identifiers** generated in Go, never by the database.
- **UTC timestamps only.** RFC 3339 text in SQLite, `timestamptz` in PostgreSQL.
- **Fractional indexing** (`position` as text) for ordering, so moving an item updates a single row.
- **Workspace isolation enforced by the database** through composite foreign keys `(x_id, workspace_id)`.
- **Single owning team per task.** Multi-team work is split into subtasks via `parent_id`.
- **No ORM (Object-Relational Mapping).** Plain `database/sql` with `modernc.org/sqlite`.
- **Separate migrations** for SQLite and PostgreSQL (`migrations/sqlite`, `migrations/postgres`).

Business rules that cannot be expressed in the schema (last owner protection, subtask cycle prevention, cleanup when a member leaves or a team is removed) are enforced in the service layer and covered by unit tests.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Frontend | React, TypeScript, Vite, Tailwind CSS, Bun (package manager) |
| Backend (local) | Go, SQLite |
| Server (team) | Go, PostgreSQL |

## Repository Layout

- `apps/web` - React + TypeScript frontend
- `backend` - Go backend (configuration, database connection, embedded migrations, HTTP handlers)
