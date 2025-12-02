-- ============================================
-- Project Board Management System
-- Schema Rollback Script
-- ============================================
-- This script completely removes all tables, indexes, triggers, and functions
-- created by the init schema migration

-- ============================================
-- Drop all triggers first
-- ============================================
DROP TRIGGER IF EXISTS trigger_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS trigger_boards_updated_at ON boards;
DROP TRIGGER IF EXISTS trigger_field_options_updated_at ON field_options;
DROP TRIGGER IF EXISTS trigger_participants_updated_at ON participants;
DROP TRIGGER IF EXISTS trigger_comments_updated_at ON comments;
DROP TRIGGER IF EXISTS trigger_project_join_requests_updated_at ON project_join_requests;

-- ============================================
-- Drop tables in reverse dependency order
-- ============================================
-- Drop child tables first (those with foreign keys)
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS participants CASCADE;
DROP TABLE IF EXISTS project_join_requests CASCADE;
DROP TABLE IF EXISTS project_members CASCADE;
DROP TABLE IF EXISTS field_options CASCADE;
DROP TABLE IF EXISTS boards CASCADE;

-- Drop parent table last
DROP TABLE IF EXISTS projects CASCADE;

-- ============================================
-- Drop functions
-- ============================================
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- ============================================
-- Drop UUID extension (optional - comment out if shared with other schemas)
-- ============================================
-- DROP EXTENSION IF EXISTS "pgcrypto";

-- ============================================
-- Confirmation message
-- ============================================
DO
$$
BEGIN
    RAISE
NOTICE 'Project Board Management System schema has been completely removed.';
    RAISE
NOTICE 'All tables, triggers, and functions have been dropped.';
END $$;