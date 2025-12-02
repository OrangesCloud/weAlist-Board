-- ============================================
-- Project Board Management System
-- Migration 005 Rollback: Remove project_id from field_options
-- ============================================

-- ============================================
-- Step 1: Drop new unique constraint
-- ============================================
ALTER TABLE field_options DROP CONSTRAINT IF EXISTS uq_field_options_project_type_value;

-- ============================================
-- Step 2: Drop index
-- ============================================
DROP INDEX IF EXISTS idx_field_options_project_id;

-- ============================================
-- Step 3: Drop foreign key constraint
-- ============================================
ALTER TABLE field_options DROP CONSTRAINT IF EXISTS fk_field_options_project;

-- ============================================
-- Step 4: Drop project_id column
-- ============================================
ALTER TABLE field_options DROP COLUMN IF EXISTS project_id;

-- ============================================
-- Step 5: Restore old unique constraint
-- ============================================
ALTER TABLE field_options
ADD CONSTRAINT uq_field_options_type_value UNIQUE (field_type, value);

