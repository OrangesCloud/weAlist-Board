-- ============================================
-- Project Board Management System
-- Migration 002: Add Project Members, Join Requests, and Board Fields
-- ============================================

-- ============================================
-- Add new columns to projects table
-- ============================================
ALTER TABLE projects ADD COLUMN IF NOT EXISTS owner_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);

-- ============================================
-- Table: project_members
-- ============================================
CREATE TABLE IF NOT EXISTS project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role_name VARCHAR(50) NOT NULL CHECK (role_name IN ('OWNER', 'ADMIN', 'MEMBER')),
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_project_members_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT uq_project_members_project_user UNIQUE (project_id, user_id)
);

-- Indexes for project_members table
CREATE INDEX idx_project_members_project_id ON project_members(project_id);
CREATE INDEX idx_project_members_user_id ON project_members(user_id);
CREATE INDEX idx_project_members_role ON project_members(role_name);

-- ============================================
-- Table: project_join_requests
-- ============================================
CREATE TABLE IF NOT EXISTS project_join_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_project_join_requests_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Indexes for project_join_requests table
CREATE INDEX idx_project_join_requests_project_id ON project_join_requests(project_id);
CREATE INDEX idx_project_join_requests_user_id ON project_join_requests(user_id);
CREATE INDEX idx_project_join_requests_status ON project_join_requests(status);
CREATE INDEX idx_project_join_requests_project_status ON project_join_requests(project_id, status);

-- Trigger for project_join_requests table
CREATE TRIGGER trigger_project_join_requests_updated_at
    BEFORE UPDATE ON project_join_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Add new columns to boards table
-- ============================================
ALTER TABLE boards ADD COLUMN IF NOT EXISTS author_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
ALTER TABLE boards ADD COLUMN IF NOT EXISTS assignee_id UUID;
ALTER TABLE boards ADD COLUMN IF NOT EXISTS due_date TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_boards_author_id ON boards(author_id);
CREATE INDEX IF NOT EXISTS idx_boards_assignee_id ON boards(assignee_id);
CREATE INDEX IF NOT EXISTS idx_boards_due_date ON boards(due_date);

-- ============================================
-- Comments for documentation
-- ============================================

COMMENT ON TABLE project_members IS 'Members of projects with their roles (OWNER, ADMIN, MEMBER)';
COMMENT ON TABLE project_join_requests IS 'Requests to join projects with approval workflow';

COMMENT ON COLUMN projects.owner_id IS 'Reference to user who owns the project';
COMMENT ON COLUMN projects.is_public IS 'Whether the project is publicly visible';
COMMENT ON COLUMN project_members.role_name IS 'Member role: OWNER, ADMIN, MEMBER';
COMMENT ON COLUMN project_join_requests.status IS 'Request status: PENDING, APPROVED, REJECTED';
COMMENT ON COLUMN boards.author_id IS 'Reference to user who created the board';
COMMENT ON COLUMN boards.assignee_id IS 'Reference to user assigned to the board';
COMMENT ON COLUMN boards.due_date IS 'Due date for the board task';
