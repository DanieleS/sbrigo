-- Liste multiple: ogni task appartiene a una lista; la lista determina il tipo (spesa o attività).
CREATE TABLE lists (
    id          UUID PRIMARY KEY,
    name        VARCHAR(120) NOT NULL,
    kind        VARCHAR(32) NOT NULL CHECK (kind IN ('grocery', 'general_task')),
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

INSERT INTO lists (id, name, kind, sort_order) VALUES
    (gen_random_uuid(), 'Spesa', 'grocery', 10),
    (gen_random_uuid(), 'Casa',  'general_task', 10);

ALTER TABLE tasks ADD COLUMN list_id UUID REFERENCES lists(id) ON DELETE CASCADE;
UPDATE tasks t SET list_id = l.id FROM lists l WHERE l.kind = t.item_type;
ALTER TABLE tasks ALTER COLUMN list_id SET NOT NULL;

CREATE INDEX tasks_list_id_idx ON tasks (list_id, is_completed);
