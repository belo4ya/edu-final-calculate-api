-- Users table
CREATE TABLE users
(
    id            TEXT PRIMARY KEY,
    login         TEXT      NOT NULL UNIQUE,
    password_hash TEXT      NOT NULL,

    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Expressions table
CREATE TABLE expressions
(
    id         TEXT PRIMARY KEY,
    user_id    TEXT      NOT NULL,
    expression TEXT      NOT NULL,
    status     TEXT      NOT NULL,
    result     REAL,
    error      TEXT,

    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE INDEX idx_expressions_user_id ON expressions (user_id);
CREATE INDEX idx_expressions_user_status ON expressions (user_id, status);

-- Tasks table
CREATE TABLE tasks
(
    id               TEXT PRIMARY KEY,
    expression_id    TEXT      NOT NULL,
    parent_task_1_id TEXT,
    parent_task_2_id TEXT,
    arg1             REAL      NOT NULL,
    arg2             REAL      NOT NULL,
    operation        TEXT      NOT NULL,
    operation_time   INTEGER   NOT NULL, -- stored in milliseconds
    status           TEXT      NOT NULL,
    result           REAL,
    expire_at        TIMESTAMP,

    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (expression_id) REFERENCES expressions (id) ON DELETE CASCADE,
    FOREIGN KEY (parent_task_1_id) REFERENCES tasks (id),
    FOREIGN KEY (parent_task_2_id) REFERENCES tasks (id)
);

CREATE INDEX idx_tasks_expression_id ON tasks (expression_id);
CREATE INDEX idx_tasks_parent_tasks ON tasks (parent_task_1_id, parent_task_2_id);
CREATE INDEX idx_tasks_status_created ON tasks (status, created_at);
