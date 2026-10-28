CREATE TABLE IF NOT EXISTS "teams" (
    "id" TEXT NOT NULL,
    "workspace_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "color" TEXT,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "deleted_at" DATETIME,
    PRIMARY KEY ("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS "teams_idx_teams_id_workspace_id"
ON "teams" ("id", "workspace_id");
CREATE UNIQUE INDEX IF NOT EXISTS "teams_idx_teams_workspace_id_name"
ON "teams" ("workspace_id", "name");

CREATE TABLE IF NOT EXISTS "workspaces" (
    "id" TEXT NOT NULL,
    -- personal | team
    "kind" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "deleted_at" DATETIME,
    PRIMARY KEY ("id")
);


CREATE TABLE IF NOT EXISTS "tasks" (
    "id" TEXT NOT NULL,
    "created_by" TEXT NOT NULL,
    "project_id" TEXT NOT NULL,
    "workspace_id" TEXT NOT NULL,
    "status_id" TEXT NOT NULL,
    "parent_id" TEXT,
    "title" TEXT NOT NULL,
    "description" TEXT,
    -- fractional index, status icindeki sıra
    "position" TEXT NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "deleted_at" DATETIME,
    PRIMARY KEY ("id"),
    FOREIGN KEY("created_by") REFERENCES "memberships"("id"),
    FOREIGN KEY("parent_id") REFERENCES "tasks"("id"),
    FOREIGN KEY("project_id") REFERENCES "projects"("id"),
    FOREIGN KEY("status_id") REFERENCES "statuses"("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);

-- Indexes
CREATE INDEX IF NOT EXISTS "tasks_idx_tasks_project_id_status_id_position"
ON "tasks" ("project_id", "status_id", "position");
CREATE INDEX IF NOT EXISTS "tasks_idx_tasks_workspace_id_updated_at"
ON "tasks" ("workspace_id", "updated_at");

CREATE TABLE IF NOT EXISTS "invitations" (
    "id" TEXT NOT NULL,
    "workspace_id" TEXT NOT NULL,
    "invited_by" TEXT NOT NULL,
    -- doluysa davet bu e-postaya bağlı
    "email" TEXT,
    "role" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL UNIQUE,
    "expires_at" DATETIME NOT NULL,
    "accepted_at" DATETIME,
    "revoked_at" DATETIME,
    "created_at" DATETIME NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY("invited_by") REFERENCES "memberships"("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);


CREATE TABLE IF NOT EXISTS "project_teams" (
    "workspace_id" TEXT NOT NULL,
    "project_id" TEXT NOT NULL,
    "team_id" TEXT NOT NULL,
    "added_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "removed_at" DATETIME,
    PRIMARY KEY ("project_id", "team_id"),
    FOREIGN KEY("project_id") REFERENCES "projects"("id"),
    FOREIGN KEY("team_id") REFERENCES "teams"("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);


CREATE TABLE IF NOT EXISTS "team_members" (
    "workspace_id" TEXT NOT NULL,
    "membership_id" TEXT NOT NULL,
    "team_id" TEXT NOT NULL,
    -- lead | member
    "role" TEXT NOT NULL,
    "added_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "removed_at" DATETIME,
    PRIMARY KEY ("membership_id", "team_id"),
    FOREIGN KEY("membership_id") REFERENCES "memberships"("id"),
    FOREIGN KEY("team_id") REFERENCES "teams"("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);


CREATE TABLE IF NOT EXISTS "users" (
    "id" TEXT NOT NULL,
    "display_name" TEXT NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY("id") REFERENCES "accounts"("user_id")
);


CREATE TABLE IF NOT EXISTS "accounts" (
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL UNIQUE,
    -- kucuk harfe normalize
    "email" TEXT NOT NULL UNIQUE,
    "password_hash" TEXT NOT NULL,
    "created_at" DATETIME NOT NULL,
    PRIMARY KEY ("id")
);


CREATE TABLE IF NOT EXISTS "projects" (
    "id" TEXT NOT NULL,
    "workspace_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "archived_at" DATETIME,
    "deleted_at" DATETIME,
    PRIMARY KEY ("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS "projects_idx_projects_id_workspace_id"
ON "projects" ("id", "workspace_id");

CREATE TABLE IF NOT EXISTS "statuses" (
    "id" TEXT NOT NULL,
    "workspace_id" TEXT NOT NULL,
    "project_id" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "color" TEXT NOT NULL,
    -- fractional index
    "position" TEXT NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "deleted_at" DATETIME,
    PRIMARY KEY ("id"),
    FOREIGN KEY("project_id") REFERENCES "projects"("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);

-- Indexes
CREATE INDEX IF NOT EXISTS "statuses_idx_statuses_project_id_position"
ON "statuses" ("project_id", "position");
CREATE UNIQUE INDEX IF NOT EXISTS "statuses_idx_statuses_id_workspace_id"
ON "statuses" ("id", "workspace_id");

CREATE TABLE IF NOT EXISTS "task_assignees" (
    "workspace_id" TEXT NOT NULL,
    "task_id" TEXT NOT NULL,
    "membership_id" TEXT NOT NULL,
    "assigned_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "removed_at" DATETIME,
    PRIMARY KEY ("task_id", "membership_id"),
    FOREIGN KEY("membership_id") REFERENCES "memberships"("id"),
    FOREIGN KEY("task_id") REFERENCES "tasks"("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);


CREATE TABLE IF NOT EXISTS "memberships" (
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "workspace_id" TEXT NOT NULL,
    -- owner | admin | member | viewer
    "role" TEXT NOT NULL,
    "joined_at" DATETIME NOT NULL,
    "left_at" DATETIME,
    "updated_at" DATETIME NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY("user_id") REFERENCES "users"("id"),
    FOREIGN KEY("workspace_id") REFERENCES "workspaces"("id")
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS "memberships_idx_memberships_workspace_id_user_id"
ON "memberships" ("workspace_id", "user_id");
CREATE UNIQUE INDEX IF NOT EXISTS "memberships_index_3"
ON "memberships" ("id", "workspace_id");
