-- ============================================
-- Project Board Management System
-- Migration 005: Add project_id to field_options
-- ============================================
-- This migration adds project_id to field_options table to support
-- project-specific customization of field options

-- ============================================
-- Step 1: Add project_id column (nullable first)
-- ============================================
ALTER TABLE field_options 
ADD COLUMN project_id UUID;

-- ============================================
-- Step 2: Add foreign key constraint
-- ============================================
ALTER TABLE field_options
ADD CONSTRAINT fk_field_options_project
FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

-- ============================================
-- Step 3: Create index for project_id
-- ============================================
CREATE INDEX idx_field_options_project_id ON field_options(project_id);

-- ============================================
-- Step 4: Update unique constraint to include project_id
-- ============================================
-- Drop old constraint
ALTER TABLE field_options DROP CONSTRAINT IF EXISTS uq_field_options_type_value;

-- Add new constraint that includes project_id
-- NULL project_id means system default (shared across all projects)
ALTER TABLE field_options
ADD CONSTRAINT uq_field_options_project_type_value 
UNIQUE (project_id, field_type, value);

-- ============================================
-- Comments for documentation
-- ============================================
COMMENT ON COLUMN field_options.project_id IS 'Project ID for project-specific options. NULL means system default template.';

