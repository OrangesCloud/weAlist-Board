package domain

import "github.com/google/uuid"

// FieldType represents the type of a custom field
type FieldType string

// FieldType constants
const (
	FieldTypeStage      FieldType = "stage"
	FieldTypeRole       FieldType = "role"
	FieldTypeImportance FieldType = "importance"
)

// FieldOption represents a selectable option for custom fields (stage, role, importance)
type FieldOption struct {
	BaseModel
	ProjectID       *uuid.UUID `gorm:"type:uuid;index:idx_field_options_project_id;uniqueIndex:uq_field_options_project_type_value,priority:1" json:"project_id"` // NULL for system defaults
	FieldType       FieldType  `gorm:"type:varchar(50);not null;index:idx_field_options_field_type;uniqueIndex:uq_field_options_project_type_value,priority:2" json:"field_type"`
	Value           string     `gorm:"type:varchar(100);not null;uniqueIndex:uq_field_options_project_type_value,priority:3" json:"value"`
	Label           string     `gorm:"type:varchar(200);not null" json:"label"`
	Color           string     `gorm:"type:varchar(20);not null" json:"color"`
	DisplayOrder    int        `gorm:"type:int;not null;default:0;index:idx_field_options_display_order" json:"display_order"`
	IsSystemDefault bool       `gorm:"type:boolean;not null;default:false" json:"is_system_default"`
	Project         *Project   `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"project,omitempty"`
}

// TableName specifies the table name for FieldOption
func (FieldOption) TableName() string {
	return "field_options"
}
