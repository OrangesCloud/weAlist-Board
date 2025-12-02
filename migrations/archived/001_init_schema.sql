-- ============================================
-- Project Board Management System
-- Initial Schema Migration
-- ============================================

-- Enable UUID extension (must be run as superuser)
-- This should be done during database initialization
-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- Table: projects
-- ============================================
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Indexes for projects table
CREATE INDEX idx_projects_workspace_id ON projects(workspace_id);
CREATE INDEX idx_projects_is_default ON projects(is_default);
CREATE INDEX idx_projects_deleted_at ON projects(deleted_at);
CREATE INDEX idx_projects_workspace_default ON projects(workspace_id, is_default) WHERE deleted_at IS NULL;

-- ============================================
-- Table: boards
-- ============================================
CREATE TABLE IF NOT EXISTS boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    stage VARCHAR(50) NOT NULL CHECK (stage IN ('in_progress', 'pending', 'approved', 'review')),
    importance VARCHAR(50) NOT NULL CHECK (importance IN ('urgent', 'normal')),
    role VARCHAR(50) NOT NULL CHECK (role IN ('developer', 'planner')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_boards_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Indexes for boards table
CREATE INDEX idx_boards_project_id ON boards(project_id);
CREATE INDEX idx_boards_deleted_at ON boards(deleted_at);
CREATE INDEX idx_boards_stage ON boards(stage);
CREATE INDEX idx_boards_importance ON boards(importance);
CREATE INDEX idx_boards_role ON boards(role);
CREATE INDEX idx_boards_project_active ON boards(project_id) WHERE deleted_at IS NULL;

-- ============================================
-- Table: participants
-- ============================================
CREATE TABLE IF NOT EXISTS participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_participants_board FOREIGN KEY (board_id) REFERENCES boards(id) ON DELETE CASCADE,
    CONSTRAINT uq_participants_board_user UNIQUE (board_id, user_id)
);

-- Indexes for participants table
CREATE INDEX idx_participants_board_id ON participants(board_id);
CREATE INDEX idx_participants_user_id ON participants(user_id);
CREATE INDEX idx_participants_deleted_at ON participants(deleted_at);
CREATE INDEX idx_participants_board_active ON participants(board_id) WHERE deleted_at IS NULL;

-- ============================================
-- Table: comments
-- ============================================
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_comments_board FOREIGN KEY (board_id) REFERENCES boards(id) ON DELETE CASCADE
);

-- Indexes for comments table
CREATE INDEX idx_comments_board_id ON comments(board_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_deleted_at ON comments(deleted_at);
CREATE INDEX idx_comments_board_created ON comments(board_id, created_at) WHERE deleted_at IS NULL;

-- ============================================
-- Triggers for updated_at timestamps
-- ============================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for projects table
CREATE TRIGGER trigger_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for boards table
CREATE TRIGGER trigger_boards_updated_at
    BEFORE UPDATE ON boards
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for participants table
CREATE TRIGGER trigger_participants_updated_at
    BEFORE UPDATE ON participants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for comments table
CREATE TRIGGER trigger_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Comments for documentation
-- ============================================

COMMENT ON TABLE projects IS 'Projects belong to workspaces and contain boards';
COMMENT ON TABLE boards IS 'Boards represent work items with stage, importance, and role attributes';
COMMENT ON TABLE participants IS 'Users participating in specific boards';
COMMENT ON TABLE comments IS 'Comments on boards for discussion and collaboration';

COMMENT ON COLUMN projects.workspace_id IS 'Reference to external workspace entity';
COMMENT ON COLUMN projects.is_default IS 'Indicates if this is the default project for the workspace';
COMMENT ON COLUMN boards.stage IS 'Current stage: in_progress, pending, approved, review';
COMMENT ON COLUMN boards.importance IS 'Priority level: urgent, normal';
COMMENT ON COLUMN boards.role IS 'Assigned role: developer, planner';
COMMENT ON COLUMN participants.user_id IS 'Reference to external user entity';
COMMENT ON COLUMN comments.user_id IS 'Reference to external user entity who created the comment';
