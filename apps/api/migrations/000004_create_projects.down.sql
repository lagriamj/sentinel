DROP INDEX IF EXISTS scan_targets_project_id_idx;
DROP INDEX IF EXISTS scan_targets_project_url_key;

ALTER TABLE scan_targets DROP CONSTRAINT IF EXISTS scan_targets_project_id_fkey;
ALTER TABLE scan_targets DROP COLUMN IF EXISTS project_id;

ALTER TABLE scan_targets ADD CONSTRAINT scan_targets_url_key UNIQUE (url);

DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;
