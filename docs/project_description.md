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
| Task | A unit of work in a project and status. It can have one owning team, assignees and subtasks. It is finished when explicitly marked done, independent of its status column. |
| Comment | A note a member leaves on a task. Only its author can edit or delete it. |
| Resource | A folder or file in the workspace's single resource tree. Files can be previewed, text files edited in place. Resources can be linked to tasks (many-to-many). Deleting moves an item to a trash, from which it is restored or purged for good. |
| GitHub repository | A resource folder filled with a GitHub repository the user picked after connecting their GitHub account. The folder can be renamed, moved, trashed and linked to tasks like any folder, but its files are read-only: every device fetches its own copy as the ZIP of the newest commit, and the files can be browsed and previewed but not changed. Its activity shows the commit history of the branch as a graph (merged branches as lanes joining the main line), read from GitHub when it is reachable and not stored. Separately, the user can clone the repository with git into a folder of their choice on their computer; that clone is theirs and the app never changes it, it only remembers where it is. While GitHub can be reached and automatic updates are on (a user setting), new commits are fetched in the background; without internet the copy stays as it is and is marked as possibly out of date. |
| GitHub account | The user's connection to GitHub, made with the OAuth device flow (the user enters a code on github.com). The token is kept in the operating system's keychain, never in the database, and is never sent to the browser or synchronized. |
| Resource event | One entry of the resource activity log: who created, uploaded, edited, renamed, moved, copied, trashed, restored, purged, linked or unlinked which resource, and when. It records that a file changed, not what changed; file contents have no version history. |
| User setting | A preference that follows the user to every device: autosave on/off and delay for the resource editor, hiding completed tasks or moving them to the bottom of their status on plan boards and lists, the names of the default statuses of new plans, and which notifications are shown, and whether GitHub repositories are updated automatically. Interface-only preferences stay per device. |
| Notification | A reminder shown while the app is open: a task assigned to the user is due soon (1 hour, 1 day or 1 week before its due day), a daily summary of overdue tasks, or a comment by another member. Shown in the app and, when the browser allows it, as a system notification while the app is in the background, optionally with a sound. |
| Data export | A ZIP archive of everything the user has: all live rows as `data.json` and the resource files in their folder layout. |
| Data import | Reads a data export back. Into an empty workspace it is a full restore (names and settings included); otherwise it adds what is not already there, without deleting or overwriting anything. |
| Data reset | Deletes all content of the local workspace and restores the first-run names and settings, keeping the same user and workspace ids. |

## Design Decisions

- **Soft deletion everywhere.** Rows are marked with `deleted_at`, `removed_at` or `left_at` so that deletions can be synchronized. Physical deletion happens only in a purge job.
- **UUIDv7 identifiers** generated in Go, never by the database.
- **UTC timestamps only.** RFC 3339 text in SQLite, `timestamptz` in PostgreSQL.
- **Fractional indexing** (`position` as text) for ordering, so moving an item updates a single row.
- **Workspace isolation enforced by the database** through composite foreign keys `(x_id, workspace_id)`.
- **Single owning team per task.** Multi-team work is split into subtasks via `parent_id`.
- **No ORM (Object-Relational Mapping).** Plain `database/sql` with `modernc.org/sqlite`.
- **Separate migrations** for SQLite and PostgreSQL (`migrations/sqlite`, `migrations/postgres`).
- **Workspace search** (`GET /api/search?q=`) matches the names of live projects, plans and resources and the titles and descriptions of live tasks (subtasks included; trashed resources are skipped). Matching is a case-insensitive substring test; SQLite only folds ASCII, so the backend registers a `yap_fold` SQL function (Go `strings.ToLower`) to fold both sides. It runs without an index, which is fine for a local workspace; full-text search (FTS5 / `tsvector`) can replace it if it ever gets slow.

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
