-- ============================================
-- Project Board Management System
-- Migration 005 Rollback: Remove Custom Fields from Boards
-- ============================================
-- This rollback script restores the original board structure
-- by recreating the legacy columns and restoring data from custom_fields

-- ============================================
-- Step 1: Recreate legacy columns
-- ============================================
-- Add back the stage, importance, and role columns if they don't exist
ALTER TABLE boards ADD COLUMN IF NOT EXISTS stage VARCHAR(50);
ALTER TABLE boards ADD COLUMN IF NOT EXISTS importance VARCHAR(50);
ALTER TABLE boards ADD COLUMN IF NOT EXISTS role VARCHAR(50);

-- ============================================
-- Step 2: Restore data from custom_fields to legacy columns
-- ============================================
-- Extract values from custom_fields JSONB and populate legacy columns
UPDATE boards 
SET 
    stage = custom_fields->>'stage',
    importance = custom_fields->>'importance',
    role = custom_fields->>'role'
WHERE custom_fields IS NOT NULL;

-- ============================================
-- Step 3: Add constraints back to legacy columns
-- ============================================
-- Add NOT NULL constraints (set defaults for NULL values first)
UPDATE boards SET stage = 'pending' WHERE stage IS NULL;
UPDATE boards SET importance = 'normal' WHERE importance IS NULL;
UPDATE boards SET role = 'developer' WHERE role IS NULL;

ALTER TABLE boards ALTER COLUMN stage SET NOT NULL;
ALTER TABLE boards ALTER COLUMN importance SET NOT NULL;
ALTER TABLE boards ALTER COLUMN role SET NOT NULL;

-- Add CHECK constraints
ALTER TABLE boards ADD CONSTRAINT boards_stage_check 
    CHECK (stage IN ('in_progress', 'pending', 'approved', 'review'));
ALTER TABLE boards ADD CONSTRAINT boards_importance_check 
    CHECK (importance IN ('urgent', 'normal'));
ALTER TABLE boards ADD CONSTRAINT boards_role_check 
    CHECK (role IN ('developer', 'planner'));

-- ============================================
-- Step 4: Recreate indexes for legacy columns
-- ============================================
CREATE INDEX IF NOT EXISTS idx_boards_stage ON boards(stage);
CREATE INDEX IF NOT EXISTS idx_boards_importance ON boards(importance);
CREATE INDEX IF NOT EXISTS idx_boards_role ON boards(role);

-- ============================================
-- Step 3: Drop GIN index
-- ============================================
DROP INDEX IF EXISTS idx_boards_custom_fields;

-- ============================================
-- Step 4: Drop custom_fields column
-- ============================================
ALTER TABLE boards DROP COLUMN IF EXISTS custom_fields;

-- ============================================
-- Comments
-- ============================================
-- Note: This rollback assumes the original stage, importance, and role columns
-- still exist. If they were dropped in a later migration, those columns would
-- need to be recreated first before running this rollback.
