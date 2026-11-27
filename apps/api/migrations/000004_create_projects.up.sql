-- Projects own registered targets. A scan job must refer to one of those targets.

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT projects_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT projects_name_length CHECK (char_length(name) <= 200),
    CONSTRAINT projects_description_length CHECK (char_length(description) <= 2000)
);

CREATE INDEX projects_created_by_idx ON projects (created_by);

CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id),
    CONSTRAINT project_members_role_allowed CHECK (role IN ('owner', 'member'))
);

CREATE INDEX project_members_user_id_idx ON project_members (user_id);

ALTER TABLE scan_targets
    ADD COLUMN project_id UUID REFERENCES projects (id) ON DELETE RESTRICT;

ALTER TABLE scan_targets
    ALTER COLUMN project_id SET NOT NULL;

ALTER TABLE scan_targets DROP CONSTRAINT scan_targets_url_key;

CREATE UNIQUE INDEX scan_targets_project_url_key ON scan_targets (project_id, url);

CREATE INDEX scan_targets_project_id_idx ON scan_targets (project_id);
