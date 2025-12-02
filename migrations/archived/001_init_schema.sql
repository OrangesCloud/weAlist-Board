-- ============================================
-- Project Board Management System
-- Complete Schema - Unified Migration
-- ============================================

-- Enable UUID extension (must be run as superuser)
-- This should be done during database initialization
-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- Function for updated_at timestamps
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Table: projects
-- ============================================
CREATE TABLE IF NOT EXISTS projects
(
    id           UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    workspace_id UUID         NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    is_default   BOOLEAN               DEFAULT FALSE,
    owner_id     UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    is_public    BOOLEAN               DEFAULT FALSE,
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP
);

-- Indexes for projects table
CREATE INDEX idx_projects_workspace_id ON projects (workspace_id);
CREATE INDEX idx_projects_is_default ON projects (is_default);
CREATE INDEX idx_projects_deleted_at ON projects (deleted_at);
CREATE INDEX idx_projects_workspace_default ON projects (workspace_id, is_default) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_owner_id ON projects (owner_id);

-- ============================================
-- Table: boards (with custom_fields - final version)
-- ============================================
CREATE TABLE IF NOT EXISTS boards
(
    id            UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    project_id    UUID         NOT NULL,
    title         VARCHAR(255) NOT NULL,
    content       TEXT,
    author_id     UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    assignee_id   UUID,
    due_date      TIMESTAMP,
    custom_fields JSONB                 DEFAULT '{}'::jsonb,
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMP,
    CONSTRAINT fk_boards_project FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE
);

-- Indexes for boards table
CREATE INDEX idx_boards_project_id ON boards (project_id);
CREATE INDEX idx_boards_deleted_at ON boards (deleted_at);
CREATE INDEX idx_boards_project_active ON boards (project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_boards_author_id ON boards (author_id);
CREATE INDEX idx_boards_assignee_id ON boards (assignee_id);
CREATE INDEX idx_boards_due_date ON boards (due_date);
CREATE INDEX idx_boards_custom_fields ON boards USING GIN (custom_fields);

-- ============================================
-- Table: field_options (with project_id - final version)
-- ============================================
CREATE TABLE IF NOT EXISTS field_options
(
    id                UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    field_type        VARCHAR(50)  NOT NULL CHECK (field_type IN ('stage', 'role', 'importance')),
    value             VARCHAR(100) NOT NULL,
    label             VARCHAR(200) NOT NULL,
    color             VARCHAR(20)  NOT NULL,
    display_order     INT          NOT NULL DEFAULT 0,
    is_system_default BOOLEAN      NOT NULL DEFAULT false,
    project_id        UUID,
    created_at        TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMP,

    CONSTRAINT uq_field_options_project_type_value UNIQUE (project_id, field_type, value),
    CONSTRAINT fk_field_options_project FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE
);

-- Indexes for field_options table
CREATE INDEX idx_field_options_field_type ON field_options (field_type);
CREATE INDEX idx_field_options_display_order ON field_options (display_order);
CREATE INDEX idx_field_options_deleted_at ON field_options (deleted_at);
CREATE INDEX idx_field_options_project_id ON field_options (project_id);

-- ============================================
-- Table: project_members
-- ============================================
CREATE TABLE IF NOT EXISTS project_members
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    project_id UUID        NOT NULL,
    user_id    UUID        NOT NULL,
    role_name  VARCHAR(50) NOT NULL CHECK (role_name IN ('OWNER', 'ADMIN', 'MEMBER')),
    joined_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_project_members_project FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE,
    CONSTRAINT uq_project_members_project_user UNIQUE (project_id, user_id)
);

-- Indexes for project_members table
CREATE INDEX idx_project_members_project_id ON project_members (project_id);
CREATE INDEX idx_project_members_user_id ON project_members (user_id);
CREATE INDEX idx_project_members_role ON project_members (role_name);

-- ============================================
-- Table: project_join_requests
-- ============================================
CREATE TABLE IF NOT EXISTS project_join_requests
(
    id           UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    project_id   UUID        NOT NULL,
    user_id      UUID        NOT NULL,
    status       VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    requested_at TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_project_join_requests_project FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE
);

-- Indexes for project_join_requests table
CREATE INDEX idx_project_join_requests_project_id ON project_join_requests (project_id);
CREATE INDEX idx_project_join_requests_user_id ON project_join_requests (user_id);
CREATE INDEX idx_project_join_requests_status ON project_join_requests (status);
CREATE INDEX idx_project_join_requests_project_status ON project_join_requests (project_id, status);

-- ============================================
-- Table: participants
-- ============================================
CREATE TABLE IF NOT EXISTS participants
(
    id         UUID PRIMARY KEY   DEFAULT gen_random_uuid(),
    board_id   UUID      NOT NULL,
    user_id    UUID      NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_participants_board FOREIGN KEY (board_id) REFERENCES boards (id) ON DELETE CASCADE,
    CONSTRAINT uq_participants_board_user UNIQUE (board_id, user_id)
);

-- Indexes for participants table
CREATE INDEX idx_participants_board_id ON participants (board_id);
CREATE INDEX idx_participants_user_id ON participants (user_id);
CREATE INDEX idx_participants_deleted_at ON participants (deleted_at);
CREATE INDEX idx_participants_board_active ON participants (board_id) WHERE deleted_at IS NULL;

-- ============================================
-- Table: comments
-- ============================================
CREATE TABLE IF NOT EXISTS comments
(
    id         UUID PRIMARY KEY   DEFAULT gen_random_uuid(),
    board_id   UUID      NOT NULL,
    user_id    UUID      NOT NULL,
    content    TEXT      NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_comments_board FOREIGN KEY (board_id) REFERENCES boards (id) ON DELETE CASCADE
);

-- Indexes for comments table
CREATE INDEX idx_comments_board_id ON comments (board_id);
CREATE INDEX idx_comments_user_id ON comments (user_id);
CREATE INDEX idx_comments_deleted_at ON comments (deleted_at);
CREATE INDEX idx_comments_board_created ON comments (board_id, created_at) WHERE deleted_at IS NULL;

-- ============================================
-- Triggers for updated_at timestamps
-- ============================================

-- Trigger for projects table
CREATE TRIGGER trigger_projects_updated_at
    BEFORE UPDATE
    ON projects
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for boards table
CREATE TRIGGER trigger_boards_updated_at
    BEFORE UPDATE
    ON boards
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for field_options table
CREATE TRIGGER trigger_field_options_updated_at
    BEFORE UPDATE
    ON field_options
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for participants table
CREATE TRIGGER trigger_participants_updated_at
    BEFORE UPDATE
    ON participants
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for comments table
CREATE TRIGGER trigger_comments_updated_at
    BEFORE UPDATE
    ON comments
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for project_join_requests table
CREATE TRIGGER trigger_project_join_requests_updated_at
    BEFORE UPDATE
    ON project_join_requests
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Seed Data: System Default Field Options
-- ============================================

-- Insert seed data for Stage options (system defaults - project_id is NULL)
INSERT INTO field_options (field_type, value, label, color, display_order, is_system_default)
VALUES ('stage', 'pending', '대기', '#F59E0B', 1, true),
       ('stage', 'in_progress', '진행중', '#3B82F6', 2, true),
       ('stage', 'review', '검토', '#8B5CF6', 3, true),
       ('stage', 'approved', '완료', '#10B981', 4, true);

-- Insert seed data for Role options (system defaults - project_id is NULL)
INSERT INTO field_options (field_type, value, label, color, display_order, is_system_default)
VALUES ('role', 'developer', '개발자', '#8B5CF6', 1, true),
       ('role', 'planner', '기획자', '#EC4899', 2, true);

-- Insert seed data for Importance options (system defaults - project_id is NULL)
INSERT INTO field_options (field_type, value, label, color, display_order, is_system_default)
VALUES ('importance', 'urgent', '긴급', '#EF4444', 1, true),
       ('importance', 'normal', '보통', '#10B981', 2, true);

-- ============================================
-- Comments for documentation
-- ============================================

COMMENT ON TABLE projects IS 'Projects belong to workspaces and contain boards';
COMMENT ON TABLE boards IS 'Boards represent work items with custom fields stored in JSONB format';
COMMENT ON TABLE field_options IS 'Configurable options for board fields (stage, role, importance) - can be system defaults or project-specific';
COMMENT ON TABLE project_members IS 'Members of projects with their roles (OWNER, ADMIN, MEMBER)';
COMMENT ON TABLE project_join_requests IS 'Requests to join projects with approval workflow';
COMMENT ON TABLE participants IS 'Users participating in specific boards';
COMMENT ON TABLE comments IS 'Comments on boards for discussion and collaboration';

COMMENT ON COLUMN projects.workspace_id IS 'Reference to external workspace entity';
COMMENT ON COLUMN projects.is_default IS 'Indicates if this is the default project for the workspace';
COMMENT ON COLUMN projects.owner_id IS 'Reference to user who owns the project';
COMMENT ON COLUMN projects.is_public IS 'Whether the project is publicly visible';
COMMENT ON COLUMN boards.author_id IS 'Reference to user who created the board';
COMMENT ON COLUMN boards.assignee_id IS 'Reference to user assigned to the board';
COMMENT ON COLUMN boards.due_date IS 'Due date for the board task';
COMMENT ON COLUMN boards.custom_fields IS 'JSONB field storing custom board attributes (stage, role, importance, etc.)';
COMMENT ON COLUMN field_options.field_type IS 'Type of field: stage, role, or importance';
COMMENT ON COLUMN field_options.value IS 'Enum value used in code (e.g., in_progress)';
COMMENT ON COLUMN field_options.label IS 'Display label shown to users (e.g., 진행중)';
COMMENT ON COLUMN field_options.color IS 'HEX color code for UI display (e.g., #3B82F6)';
COMMENT ON COLUMN field_options.display_order IS 'Order in which options are displayed';
COMMENT ON COLUMN field_options.is_system_default IS 'Whether this is a system default option (cannot be deleted)';
COMMENT ON COLUMN field_options.project_id IS 'Project ID for project-specific options. NULL means system default template.';
COMMENT ON COLUMN project_members.role_name IS 'Member role: OWNER, ADMIN, MEMBER';
COMMENT ON COLUMN project_join_requests.status IS 'Request status: PENDING, APPROVED, REJECTED';
COMMENT ON COLUMN participants.user_id IS 'Reference to external user entity';
COMMENT ON COLUMN comments.user_id IS 'Reference to external user entity who created the comment';