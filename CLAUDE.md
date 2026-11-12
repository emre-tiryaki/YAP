# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

YAP (Yet Another Planner) is a local-first personal/team project planner. Today only the local stack exists: React frontend (`apps/web`) → Go backend (`backend`) → SQLite. A future Go server with PostgreSQL will handle teams and sync; the schema is designed for that now.

Read before non-trivial changes:
- `docs/conventions.md` — binding coding rules for backend and frontend (rules marked **(DB)** must not be broken without updating `docs/database.md` first).
- `docs/database.md` — schema graph, consistency rules (DB-enforced vs. service-enforced), indexes, sync notes, out-of-scope list.
- `docs/project_description.md` — domain model (User, Workspace, Membership, Team, Project, Status, Task) and design decisions.

## Commands

A `.env` at the repo root is required (`DB_PATH`, `BACKEND_PORT`, `COOKIE_SECURE`). The backend finds it by walking up from cwd (stopping at `.git`) and resolves relative `DB_PATH` against the `.env` directory. The Vite config throws if `../../.env` or `BACKEND_PORT` is missing.

```bash
make dev                                  # start backend, wait for /hello-world, then start Vite

# Backend (run from backend/)
go run ./cmd/backend
go test -race ./...
go test ./internal/service -run TestName  # single test
go vet ./...
gofmt -l .                                # must print nothing

# Frontend (run from apps/web/, Bun only)
bun install
bun run dev
bun run lint
bun run build                             # tsc -b && vite build
```

There are no frontend tests. Before pushing: `go vet`, `go test -race`, gofmt-clean, `bun run lint`, `bun run build`.

## Backend architecture

Strict one-way layering: `handler → service → repository → database`, with `model` holding plain types and `model.NewID()` (UUIDv7).

- **Wiring** is all in `cmd/backend/main.go`: repositories → services → handlers, each handler exposes `Register(mux)` using `http.ServeMux` method+path patterns under `/api/...`.
- **Session**: there is no auth yet. At startup `SessionService.Bootstrap` creates (or loads) the single local user, their `personal` workspace and owner membership, and the resulting `model.Session` is injected into every handler. All queries are scoped by `session`'s `workspace_id`.
- **Transactions**: services own transactions via `database.WithTx`. Repositories are stateless and take a `database.DBTX` (either `*sql.DB` or `*sql.Tx`) as a parameter on every method — they never open transactions themselves. Service-side repository interfaces are declared in the `service` package (consumer-defined).
- **Errors**: sentinel errors live in `internal/service/errors.go`; handlers map them to status codes with `errors.Is` and never leak internal error text. JSON helpers are in `internal/handler/json.go`.
- **Ordering**: `internal/position` implements fractional-index strings for `position` columns (compare with plain byte order, never locale-aware).
- **Time**: injected via `internal/service/clock.go`; always UTC RFC 3339 strings.

### Database / migrations

- Pure-Go driver `modernc.org/sqlite`, plain `database/sql` (no ORM, no query builder). Pragmas: `foreign_keys`, WAL, `busy_timeout`.
- Migrations are embedded from `internal/database/migrations/sqlite/NNN_*.sql` and applied in order on `database.New`; progress is tracked with `PRAGMA user_version` = number of applied files. Add new migrations as new files — never edit a merged one, never reorder.
- Tables are `STRICT`. Synchronized tables carry `workspace_id`, `updated_at`, and composite FKs `(x_id, workspace_id)` for workspace isolation.
- Soft delete only (`deleted_at`, `removed_at`, `left_at`); reads must filter these out.

### Tests

Table-driven, against a real SQLite DB in `t.TempDir()` created via `database.New` (so migrations and pragmas match production) — do not mock the database. Each package has a `testhelper_test.go` (e.g. `service.newTestDB` returns a DB plus a bootstrapped session).

## Frontend architecture

React 19 + TypeScript (strict, `verbatimModuleSyntax`, `erasableSyntaxOnly` → no `enum`/`namespace`), Vite, Tailwind v4, TanStack Query, react-router.

- Vite proxies `/api` to `http://localhost:$BACKEND_PORT`, so the frontend calls relative `/api/...` paths.
- `src/api/client.ts` is the single HTTP boundary: it defines snake_case `*Dto` types, maps them to camelCase domain types in `src/types/`, parses dates, and throws typed `ApiError(status, code)` from the backend's `{"error": "..."}` body. Components never call `fetch`.
- Data fetching is done through TanStack Query hooks in `src/hooks/` (`useProjects`, `usePlans`, `useTasks`, `useComments`, `useWorkspace`).
- Routes (`App.tsx`): `/home` (workspace home), `/projects/:projectId` (project detail with plans), `/projects/:projectId/plans/:planId` (plan board: list/kanban views, drag-and-drop via `@dnd-kit`).
- UI strings go through `useTranslate()` keyed by `src/i18n/en.json` (`tr.json` also exists).
- Folders are organized by type: `api/`, `components/`, `hooks/`, `pages/`, `types/`, `i18n/`, `util/` (see `docs/conventions.md` §3.3).
- Component files export only components (react-refresh lint rule); named exports except route pages.

## Conventions worth remembering

- Conventional Commits (`feat:`, `fix:`, `test:`, ...), English everywhere.
- Business rules and bug fixes ship with tests in the same commit.
- JSON fields are `snake_case`; DB row structs are never encoded directly — handlers use response DTOs.
- Don't add features outside `docs/project_description.md` / check `docs/database.md` "Out of Scope" first.

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
