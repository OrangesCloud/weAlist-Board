-- ============================================
-- Project Board Management System
-- Migration 003 Rollback: Remove Migrated Project Owners
-- ============================================

-- ============================================
-- Remove OWNER members that were created by migration
-- ============================================
-- This rollback removes OWNER project_members entries that were created
-- during the migration. It only removes entries where the user_id matches
-- the project's owner_id to avoid removing manually added members.

DELETE FROM project_members
WHERE role_name = 'OWNER'
  AND EXISTS (
    SELECT 1 
    FROM projects p 
    WHERE p.id = project_members.project_id 
      AND p.owner_id = project_members.user_id
  );
