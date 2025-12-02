package service

import "project-board-api/internal/domain"

// fieldOptionTemplate represents a template for creating field options
type fieldOptionTemplate struct {
	FieldType    domain.FieldType
	Value        string
	Label        string
	Color        string
	DisplayOrder int
}

// getDefaultFieldOptions returns hardcoded default field options
// These options are created for every new project
// Returns 12 options total: 4 stage, 4 importance, 4 role
func getDefaultFieldOptions() []fieldOptionTemplate {
	return []fieldOptionTemplate{
		// Stage options (4개)
		{FieldType: domain.FieldTypeStage, Value: "pending", Label: "대기", Color: "#F59E0B", DisplayOrder: 1},
		{FieldType: domain.FieldTypeStage, Value: "in_progress", Label: "진행중", Color: "#3B82F6", DisplayOrder: 2},
		{FieldType: domain.FieldTypeStage, Value: "review", Label: "검토", Color: "#8B5CF6", DisplayOrder: 3},
		{FieldType: domain.FieldTypeStage, Value: "approved", Label: "완료", Color: "#10B981", DisplayOrder: 4},

		// Importance options (4개)
		{FieldType: domain.FieldTypeImportance, Value: "urgent", Label: "긴급", Color: "#EF4444", DisplayOrder: 1},
		{FieldType: domain.FieldTypeImportance, Value: "high", Label: "높음", Color: "#F97316", DisplayOrder: 2},
		{FieldType: domain.FieldTypeImportance, Value: "normal", Label: "보통", Color: "#10B981", DisplayOrder: 3},
		{FieldType: domain.FieldTypeImportance, Value: "low", Label: "낮음", Color: "#6B7280", DisplayOrder: 4},

		// Role options (4개)
		{FieldType: domain.FieldTypeRole, Value: "developer", Label: "개발자", Color: "#8B5CF6", DisplayOrder: 1},
		{FieldType: domain.FieldTypeRole, Value: "planner", Label: "기획자", Color: "#EC4899", DisplayOrder: 2},
		{FieldType: domain.FieldTypeRole, Value: "designer", Label: "디자이너", Color: "#F59E0B", DisplayOrder: 3},
		{FieldType: domain.FieldTypeRole, Value: "qa", Label: "QA", Color: "#06B6D4", DisplayOrder: 4},
	}
}
