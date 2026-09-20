-- Ordine manuale degli elementi dentro un reparto/lista (drag and drop). Gli elementi esistenti
-- mantengono l'ordine di creazione.
ALTER TABLE tasks ADD COLUMN position DOUBLE PRECISION NOT NULL DEFAULT 0;
UPDATE tasks SET position = EXTRACT(EPOCH FROM created_at) * 1000;
