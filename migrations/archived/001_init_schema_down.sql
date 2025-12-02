-- ============================================
-- Project Board Management System
-- Rollback Initial Schema Migration
-- ============================================

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_comments_updated_at ON comments;
DROP TRIGGER IF EXISTS trigger_participants_updated_at ON participants;
DROP TRIGGER IF EXISTS trigger_boards_updated_at ON boards;
DROP TRIGGER IF EXISTS trigger_projects_updated_at ON projects;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS participants CASCADE;
DROP TABLE IF EXISTS boards CASCADE;
DROP TABLE IF EXISTS projects CASCADE;

-- Note: We don't drop the pgcrypto extension as it might be used by other schemas
-- DROP EXTENSION IF EXISTS "pgcrypto";
