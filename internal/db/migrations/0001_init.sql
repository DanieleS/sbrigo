-- Schema iniziale di Sbrigo (vedi docs/REQUISITI_E_ARCHITETTURA.md, sezione 7).

CREATE TABLE users (
    id            UUID PRIMARY KEY,
    identity_id   VARCHAR(255) NOT NULL UNIQUE,
    email         VARCHAR(320) NOT NULL DEFAULT '',
    display_name  VARCHAR(255) NOT NULL DEFAULT '',
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE departments (
    id                  UUID PRIMARY KEY,
    name                VARCHAR(120) NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    default_sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE supermarkets (
    id    UUID PRIMARY KEY,
    name  VARCHAR(120) NOT NULL
);

CREATE TABLE supermarket_department_orders (
    supermarket_id  UUID NOT NULL REFERENCES supermarkets(id) ON DELETE CASCADE,
    department_id   UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    sort_order      INTEGER NOT NULL,
    PRIMARY KEY (supermarket_id, department_id)
);

CREATE TABLE tasks (
    id             UUID PRIMARY KEY,
    title          VARCHAR(500) NOT NULL,
    notes          TEXT,
    is_completed   BOOLEAN NOT NULL DEFAULT false,
    item_type      VARCHAR(32) NOT NULL CHECK (item_type IN ('grocery', 'general_task')),
    department_id  UUID REFERENCES departments(id) ON DELETE SET NULL,
    assignee_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date       TIMESTAMP WITH TIME ZONE,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX tasks_item_type_completed_idx ON tasks (item_type, is_completed);
CREATE INDEX tasks_updated_at_idx ON tasks (updated_at);
CREATE INDEX tasks_department_id_idx ON tasks (department_id);
CREATE INDEX tasks_assignee_id_idx ON tasks (assignee_id);
