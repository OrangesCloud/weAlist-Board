-- ============================================
-- Project Board Management System
-- Migration 004: Add Field Options Table
-- ============================================

-- ============================================
-- Table: field_options
-- ============================================
-- This table stores configurable options for board fields (stage, role, importance)
-- Users can customize these options while system defaults are protected

CREATE TABLE IF NOT EXISTS field_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_type VARCHAR(50) NOT NULL CHECK (field_type IN ('stage', 'role', 'importance')),
    value VARCHAR(100) NOT NULL,
    label VARCHAR(200) NOT NULL,
    color VARCHAR(20) NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    is_system_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    
    CONSTRAINT uq_field_options_type_value UNIQUE (field_type, value)
);

-- ============================================
-- Indexes for field_options table
-- ============================================
CREATE INDEX idx_field_options_field_type ON field_options(field_type);
CREATE INDEX idx_field_options_display_order ON field_options(display_order);
CREATE INDEX idx_field_options_deleted_at ON field_options(deleted_at);

-- ============================================
-- Trigger for updated_at
-- ============================================
CREATE TRIGGER trigger_field_options_updated_at
    BEFORE UPDATE ON field_options
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Insert seed data for Stage options
-- ============================================
INSERT INTO field_options (field_type, value, label, color, display_order, is_system_default) VALUES
('stage', 'pending', '대기', '#F59E0B', 1, true),
('stage', 'in_progress', '진행중', '#3B82F6', 2, true),
('stage', 'review', '검토', '#8B5CF6', 3, true),
('stage', 'approved', '완료', '#10B981', 4, true);

-- ============================================
-- Insert seed data for Role options
-- ============================================
INSERT INTO field_options (field_type, value, label, color, display_order, is_system_default) VALUES
('role', 'developer', '개발자', '#8B5CF6', 1, true),
('role', 'planner', '기획자', '#EC4899', 2, true);

-- ============================================
-- Insert seed data for Importance options
-- ============================================
INSERT INTO field_options (field_type, value, label, color, display_order, is_system_default) VALUES
('importance', 'urgent', '긴급', '#EF4444', 1, true),
('importance', 'normal', '보통', '#10B981', 2, true);

-- ============================================
-- Comments for documentation
-- ============================================
COMMENT ON TABLE field_options IS 'Configurable options for board fields (stage, role, importance)';
COMMENT ON COLUMN field_options.field_type IS 'Type of field: stage, role, or importance';
COMMENT ON COLUMN field_options.value IS 'Enum value used in code (e.g., in_progress)';
COMMENT ON COLUMN field_options.label IS 'Display label shown to users (e.g., 진행중)';
COMMENT ON COLUMN field_options.color IS 'HEX color code for UI display (e.g., #3B82F6)';
COMMENT ON COLUMN field_options.display_order IS 'Order in which options are displayed';
COMMENT ON COLUMN field_options.is_system_default IS 'Whether this is a system default option (cannot be deleted)';
