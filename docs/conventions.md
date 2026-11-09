# YAP Development Conventions

These rules apply to all code in `apps/` and `backend/`. Rules marked **(DB)** come from the database design document and are not negotiable without updating that document first.

## 1. General

| Topic | Rule |
| ----- | ---- |
| Language | Code, identifiers, comments, commit messages and docs are written in English. |
| Comments | Only for non-obvious logic (why, not what). No commented-out code. |
| Commits | Conventional Commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`). One logical change per commit. |
| Tests | Business rules and bug fixes ship with tests in the same commit. |
| Scope | No feature is added that is not in the project description or the "Out of Scope" list is explicitly revisited first. |

## 2. Backend (Go)

### 2.1 Layering

Dependency direction is one-way: `handler -> service -> repository -> database`.

| Layer | Responsibility | Must not |
| ----- | -------------- | -------- |
| `handler` | HTTP (Hypertext Transfer Protocol) parsing, validation of input shape, status codes, JSON encoding | Contain business rules or SQL (Structured Query Language) |
| `service` | Business rules, transactions, authorization checks | Know about `http.Request` or `http.ResponseWriter` |
| `repository` | SQL and row mapping only | Contain business rules or open transactions on its own |
| `model` | Plain domain types, `NewID()` | Import any other internal package |

- Services depend on interfaces defined by the consumer (the service package), not on concrete repository types.
- Dependencies are passed through constructors. No package-level mutable state, no `init()` with side effects.
- Every function that touches the database or does I/O takes `context.Context` as its first parameter.

### 2.2 Database access

- Plain `database/sql`. No ORM (Object-Relational Mapping), no query builders. **(DB)**
- Every query is parameterized. String concatenation into SQL is forbidden.
- Every query on a synchronized table filters by `workspace_id`.
- Reads exclude soft-deleted rows: `deleted_at IS NULL` for entities, `removed_at IS NULL` for relationship rows, `left_at IS NULL` for memberships.
- Deletion is soft only. Physical `DELETE` appears only in the purge job. **(DB)**
- Re-adding a removed relationship clears `removed_at` instead of inserting a new row. **(DB)**
- Every write sets `updated_at`.
- A rule that touches several rows (member leaves, team removed, project deleted) runs inside a single transaction. **(DB)**
- Always `defer rows.Close()` and check `rows.Err()` after iteration.
- Do not rely on `ON DELETE` behavior; it is intentionally unspecified. **(DB)**

### 2.3 Data conventions

| Topic | Rule |
| ----- | ---- |
| IDs | UUIDv7 (Universally Unique Identifier version 7) from `model.NewID()`. The database never generates IDs. **(DB)** |
| Time | Always UTC. Write with `time.Now().UTC().Format(time.RFC3339)`. Never store local time. **(DB)** |
| Ordering | `position` is a fractional-index string. Moving an item updates one row. **(DB)** |
| Enums | Go typed string constants mirroring the `CHECK` constraint values. Unknown values are rejected at the handler boundary, not only by the database. |
| Email | Lowercased and trimmed before storage. **(DB)** |
| Secrets | API keys and tokens never go into the database or logs. API keys live in the OS (operating system) keychain. **(DB)** |

### 2.4 Migrations

- Files are named `NNN_description.sql` with zero-padded sequence numbers (`001_init.sql`, `002_...`).
- A migration that has been merged is never edited. Fixes go into a new file.
- Every SQLite migration has a PostgreSQL counterpart in `migrations/postgres` once the server exists; the logical schema stays identical.
- SQLite tables are `STRICT`. **(DB)**
- Every new synchronized table carries `workspace_id` and `updated_at`, plus the composite foreign keys and unique `(id, workspace_id)` index needed for workspace isolation. **(DB)**

### 2.5 Errors and logging

- Wrap errors with context using `%w`. Error strings are lowercase, without trailing punctuation: `fmt.Errorf("create db dir: %w", err)`.
- Define sentinel errors (`ErrNotFound`, `ErrLastOwner`, `ErrCycle`, ...) in the service package; handlers map them to status codes with `errors.Is`.
- Never ignore an error silently. If ignoring is intentional, say why in a comment.
- Handlers never leak internal error text to the client.
- Use `log/slog` with structured fields once the service layer exists. Do not log request bodies.

### 2.6 HTTP

- Use `http.ServeMux` method-and-path patterns (`"GET /projects/{id}"`).
- Every `http.Server` sets `ReadHeaderTimeout` (already done) and request bodies are size-limited with `http.MaxBytesReader`.
- JSON field names are `snake_case` and match the column names where a field maps directly to a column.
- Handlers translate request DTOs (Data Transfer Objects) into service calls; database row structs are never encoded directly to the response.

### 2.7 Business rules (service layer)

The following are enforced in the service layer and each has a unit test **(DB)**:

- One `personal` workspace per user; no teams in a `personal` workspace.
- The last `owner` cannot leave or be downgraded.
- A member who has left cannot be assigned to a team or task; leaving closes `team_members` and `task_assignees` rows in the same transaction.
- Subtasks cannot form a cycle.
- Removing or deleting a team sets `team_id = NULL` on its tasks within the affected project, in the same transaction.
- Deleting a project marks its `statuses`, `tasks` and `project_teams` rows individually.

### 2.8 Testing

- Table-driven tests with `t.Run`.
- Repository and service tests run against a real SQLite database in `t.TempDir()` with the production pragmas. Do not mock the database.
- Use `t.Helper()` in helpers and `t.Cleanup` for teardown.
- Run `go vet ./...` and `go test -race ./...` before every push.
- Code is `gofmt`-clean. A file with formatting diffs does not get merged.

## 3. Frontend (React + TypeScript)

### 3.1 Tooling

- Bun is the only package manager. Commit `bun.lock`; never commit `package-lock.json` or `yarn.lock`.
- `bun run lint` and `bun run build` (which runs `tsc -b`) must pass before every push.
- Dependencies are added deliberately. A new runtime dependency needs a one-line justification in the commit message.

### 3.2 TypeScript

- `strict` mode is mandatory. No `any`; use `unknown` and narrow.
- No non-null assertion (`!`) except for the existing root element lookup in `main.tsx`.
- `verbatimModuleSyntax` is on: type-only imports use `import type`.
- `erasableSyntaxOnly` is on: no `enum`, no `namespace`, no constructor parameter properties. Use union types and `as const` objects.
- Domain enums are string unions that mirror the backend: `type StatusCategory = 'todo' | 'in_progress' | 'done'`.
- IDs are `string`. Use branded types if mixing ID kinds becomes a real bug source.

### 3.3 Structure

- Type-based layout under `src/`:

| Directory | Contents |
| --------- | -------- |
| `api/` | `client.ts`: the only module that calls `fetch` |
| `components/` | Reusable UI components (`TaskCard.tsx`, `Modal.tsx`, ...) |
| `hooks/` | TanStack Query hooks per resource (`useProjects.ts`, `useTasks.ts`, ...) and `useTranslate.ts` |
| `pages/` | Route-level pages, registered in `App.tsx` |
| `types/` | Domain types per resource (`project.ts`, `task.ts`, ...) |
| `i18n/` | Translation files (`en.json`, `tr.json`) |
| `util/` | Build-time helpers (`env.ts`, read by `vite.config.ts`) |

- One component per file; the file name equals the component name (`TaskCard.tsx`).
- Files that export components export only components (required by `eslint-plugin-react-refresh`). Hooks, constants and helpers go in separate files.
- Named exports for everything except route-level pages and `App`, which use `export default`.

### 3.4 Components and state

- Function components and hooks only.
- Components render; they do not call `fetch` directly. All HTTP calls go through the typed API client in `src/api/client.ts`.
- Server state lives in TanStack Query. Components and pages use the hooks in `src/hooks/`, not the API client directly; mutations invalidate the affected query keys in `onSuccess`.
- Query keys are arrays that start with the resource name and nest by ID: `["projects"]`, `["projects", id]`, `["projects", id, "plans"]`.
- Hooks that take a route parameter disable the query while it is empty (`enabled: projectId !== ""`).
- Keep state at the lowest level that needs it. Derive values during render instead of mirroring them in state.
- `useEffect` is for synchronizing with external systems only. Fetching, derived data and event reactions do not belong in effects.
- The `react-hooks` lint rules are errors, not suggestions. Do not disable them inline without a comment explaining the case.
- Do not add `useMemo`/`useCallback` without a measured reason.
- Every list render uses a stable `key` (the entity ID), never the array index.

### 3.5 Data handling

| Topic | Rule |
| ----- | ---- |
| Dates | The API sends UTC RFC 3339 strings. Parse once at the API boundary; format to the user's local time zone only at display time. |
| Ordering | Sort by `position` with plain string comparison (`a < b`). Never use `localeCompare`; fractional indexes depend on byte order. |
| Soft deletes | Entities with `deleted_at` or `removed_at` set are never rendered. The client does not need to know how deletion is implemented. |
| Validation | The client validates for UX only. The backend is the authority. |
| DTOs | The API client declares `snake_case` `*Dto` types matching the backend JSON and maps them to `camelCase` domain types from `src/types/` with `toX()` functions. DTO types never leave `client.ts`. |
| Paths | API paths are absolute (`/api/...`). Vite proxies `/api` to the backend on `BACKEND_PORT` from the root `.env`. |
| Errors | Non-2xx responses throw `ApiError` with the HTTP `status` and the backend's `error` code (`"unknown"` if none). Components check `error instanceof ApiError` and its `status` (for example, 404 renders `NotFoundPage`). |

### 3.6 Text and translations

- User-facing strings are never hard-coded. Use `const t = useTranslate()` and `t("section.key")`.
- Keys are dot-separated by area (`sidebar.projects`, `common.close`) and defined in `src/i18n/en.json`; `TranslationKey` is derived from that file, so a missing key is a type error.
- Add every new key to `tr.json` as well.

### 3.7 Styling

- Tailwind CSS utility classes. No inline `style` unless the value is dynamic and cannot be expressed as a class.
- Repeated class combinations become a component, not a CSS file.
- Interactive elements use semantic HTML (`button`, `a`, `label`) and are keyboard-accessible.
