-- ============================================
-- Project Board Management System
-- Migration 004 Rollback: Remove Field Options Table
-- ============================================

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_field_options_updated_at ON field_options;

-- Drop indexes
DROP INDEX IF EXISTS idx_field_options_deleted_at;
DROP INDEX IF EXISTS idx_field_options_display_order;
DROP INDEX IF EXISTS idx_field_options_field_type;

-- Drop table
DROP TABLE IF EXISTS field_options;
