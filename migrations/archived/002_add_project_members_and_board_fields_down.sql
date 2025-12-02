-- ============================================
-- Project Board Management System
-- Migration 002 Rollback: Remove Project Members, Join Requests, and Board Fields
-- ============================================

-- ============================================
-- Drop new columns from boards table
-- ============================================
DROP INDEX IF EXISTS idx_boards_due_date;
DROP INDEX IF EXISTS idx_boards_assignee_id;
DROP INDEX IF EXISTS idx_boards_author_id;

ALTER TABLE boards DROP COLUMN IF EXISTS due_date;
ALTER TABLE boards DROP COLUMN IF EXISTS assignee_id;
ALTER TABLE boards DROP COLUMN IF EXISTS author_id;

-- ============================================
-- Drop project_join_requests table
-- ============================================
DROP TRIGGER IF EXISTS trigger_project_join_requests_updated_at ON project_join_requests;
DROP INDEX IF EXISTS idx_project_join_requests_project_status;
DROP INDEX IF EXISTS idx_project_join_requests_status;
DROP INDEX IF EXISTS idx_project_join_requests_user_id;
DROP INDEX IF EXISTS idx_project_join_requests_project_id;
DROP TABLE IF EXISTS project_join_requests;

-- ============================================
-- Drop project_members table
-- ============================================
DROP INDEX IF EXISTS idx_project_members_role;
DROP INDEX IF EXISTS idx_project_members_user_id;
DROP INDEX IF EXISTS idx_project_members_project_id;
DROP TABLE IF EXISTS project_members;

-- ============================================
-- Drop new columns from projects table
-- ============================================
DROP INDEX IF EXISTS idx_projects_owner_id;
ALTER TABLE projects DROP COLUMN IF EXISTS is_public;
ALTER TABLE projects DROP COLUMN IF EXISTS owner_id;
