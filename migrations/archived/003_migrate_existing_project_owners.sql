-- ============================================
-- Project Board Management System
-- Migration 003: Migrate Existing Project Owners
-- ============================================

-- ============================================
-- Create OWNER members for existing projects
-- ============================================
-- This migration creates OWNER project_members entries for all existing projects
-- that don't already have an OWNER member.
-- It uses the owner_id from the projects table.

INSERT INTO project_members (project_id, user_id, role_name, joined_at)
SELECT 
    p.id AS project_id,
    p.owner_id AS user_id,
    'OWNER' AS role_name,
    p.created_at AS joined_at
FROM projects p
WHERE p.deleted_at IS NULL
  AND p.owner_id != '00000000-0000-0000-0000-000000000000'
  AND NOT EXISTS (
    SELECT 1 
    FROM project_members pm 
    WHERE pm.project_id = p.id 
      AND pm.user_id = p.owner_id
  )
ON CONFLICT (project_id, user_id) DO NOTHING;

-- ============================================
-- Comments for documentation
-- ============================================

COMMENT ON TABLE project_members IS 'This migration ensures all existing projects have an OWNER member entry';
