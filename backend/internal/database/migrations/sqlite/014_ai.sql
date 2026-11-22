-- AI assistant: model connections, chats, messages and what was attached to
-- them. Not synchronized (see docs/database.md). API keys are never stored
-- here: has_key only says whether the OS keychain holds one.
CREATE TABLE ai_connections (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    user_id      TEXT NOT NULL REFERENCES users (id),
    name         TEXT NOT NULL CHECK (length(name) > 0),
    provider     TEXT NOT NULL CHECK (provider IN ('openai_compatible', 'anthropic')),
    preset       TEXT NOT NULL CHECK (preset IN ('ollama', 'lmstudio', 'llamacpp', 'openai', 'openrouter', 'gemini', 'groq', 'mistral', 'anthropic', 'custom')),
    base_url     TEXT NOT NULL,
    has_key      INTEGER NOT NULL DEFAULT 0 CHECK (has_key IN (0, 1)),
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT,
    UNIQUE (id, workspace_id)
) STRICT;

CREATE INDEX idx_ai_connections_user
    ON ai_connections (workspace_id, user_id) WHERE deleted_at IS NULL;

CREATE TABLE ai_chats (
    id            TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id  TEXT NOT NULL REFERENCES workspaces (id),
    user_id       TEXT NOT NULL REFERENCES users (id),
    title         TEXT NOT NULL CHECK (length(title) > 0),
    connection_id TEXT,
    model         TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    deleted_at    TEXT,
    UNIQUE (id, workspace_id),
    FOREIGN KEY (connection_id, workspace_id) REFERENCES ai_connections (id, workspace_id)
) STRICT;

CREATE INDEX idx_ai_chats_recent
    ON ai_chats (workspace_id, user_id, updated_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE ai_messages (
    id            TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id  TEXT NOT NULL,
    chat_id       TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content       TEXT NOT NULL,
    status        TEXT NOT NULL CHECK (status IN ('complete', 'streaming', 'stopped', 'interrupted', 'error')),
    error_code    TEXT,
    connection_id TEXT,
    model         TEXT,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    deleted_at    TEXT,
    UNIQUE (id, workspace_id),
    FOREIGN KEY (chat_id, workspace_id) REFERENCES ai_chats (id, workspace_id),
    FOREIGN KEY (connection_id, workspace_id) REFERENCES ai_connections (id, workspace_id)
) STRICT;

CREATE INDEX idx_ai_messages_chat
    ON ai_messages (chat_id, created_at, id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ai_messages_streaming
    ON ai_messages (status) WHERE status = 'streaming';

-- target_id is polymorphic (task, plan, project or resource id), so it has no
-- foreign key; the service checks it. Attachments live and die with their
-- message and have no soft delete of their own.
CREATE TABLE ai_message_attachments (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL,
    message_id   TEXT NOT NULL,
    position     INTEGER NOT NULL CHECK (position >= 0),
    kind         TEXT NOT NULL CHECK (kind IN ('task', 'plan', 'project', 'resource', 'repo_file')),
    target_id    TEXT NOT NULL,
    path         TEXT,
    label        TEXT NOT NULL,
    last_status  TEXT NOT NULL DEFAULT 'included' CHECK (last_status IN ('included', 'truncated', 'omitted', 'missing', 'unreadable')),
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    FOREIGN KEY (message_id, workspace_id) REFERENCES ai_messages (id, workspace_id),
    CHECK ((kind = 'repo_file') = (path IS NOT NULL))
) STRICT;

CREATE INDEX idx_ai_attachments_message
    ON ai_message_attachments (message_id, position);
