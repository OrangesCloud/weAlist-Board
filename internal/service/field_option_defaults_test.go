package service

import (
	"testing"

	"project-board-api/internal/domain"
)

func TestGetDefaultFieldOptions(t *testing.T) {
	t.Run("성공: 12개의 기본 옵션 반환", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		if len(options) != 12 {
			t.Errorf("Expected 12 default options, got %d", len(options))
		}
	})

	t.Run("성공: Stage 필드에 정확히 4개의 옵션", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		stageCount := 0
		for _, opt := range options {
			if opt.FieldType == domain.FieldTypeStage {
				stageCount++
			}
		}

		if stageCount != 4 {
			t.Errorf("Expected 4 stage options, got %d", stageCount)
		}
	})

	t.Run("성공: Importance 필드에 정확히 4개의 옵션", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		importanceCount := 0
		for _, opt := range options {
			if opt.FieldType == domain.FieldTypeImportance {
				importanceCount++
			}
		}

		if importanceCount != 4 {
			t.Errorf("Expected 4 importance options, got %d", importanceCount)
		}
	})

	t.Run("성공: Role 필드에 정확히 4개의 옵션", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		roleCount := 0
		for _, opt := range options {
			if opt.FieldType == domain.FieldTypeRole {
				roleCount++
			}
		}

		if roleCount != 4 {
			t.Errorf("Expected 4 role options, got %d", roleCount)
		}
	})

	t.Run("성공: 모든 옵션이 올바른 속성을 가짐", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		for i, opt := range options {
			// Verify FieldType is not empty
			if opt.FieldType == "" {
				t.Errorf("Option %d: FieldType is empty", i)
			}

			// Verify Value is not empty
			if opt.Value == "" {
				t.Errorf("Option %d: Value is empty", i)
			}

			// Verify Label is not empty
			if opt.Label == "" {
				t.Errorf("Option %d: Label is empty", i)
			}

			// Verify Color is not empty and starts with #
			if opt.Color == "" {
				t.Errorf("Option %d: Color is empty", i)
			} else if opt.Color[0] != '#' {
				t.Errorf("Option %d: Color should start with #, got %s", i, opt.Color)
			}

			// Verify DisplayOrder is positive
			if opt.DisplayOrder <= 0 {
				t.Errorf("Option %d: DisplayOrder should be positive, got %d", i, opt.DisplayOrder)
			}
		}
	})

	t.Run("성공: Stage 옵션의 값과 순서 검증", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		expectedStageOptions := []struct {
			value        string
			label        string
			color        string
			displayOrder int
		}{
			{"pending", "대기", "#F59E0B", 1},
			{"in_progress", "진행중", "#3B82F6", 2},
			{"review", "검토", "#8B5CF6", 3},
			{"approved", "완료", "#10B981", 4},
		}

		stageOptions := []fieldOptionTemplate{}
		for _, opt := range options {
			if opt.FieldType == domain.FieldTypeStage {
				stageOptions = append(stageOptions, opt)
			}
		}

		for i, expected := range expectedStageOptions {
			if i >= len(stageOptions) {
				t.Errorf("Missing stage option at index %d", i)
				continue
			}

			actual := stageOptions[i]
			if actual.Value != expected.value {
				t.Errorf("Stage option %d: expected value %s, got %s", i, expected.value, actual.Value)
			}
			if actual.Label != expected.label {
				t.Errorf("Stage option %d: expected label %s, got %s", i, expected.label, actual.Label)
			}
			if actual.Color != expected.color {
				t.Errorf("Stage option %d: expected color %s, got %s", i, expected.color, actual.Color)
			}
			if actual.DisplayOrder != expected.displayOrder {
				t.Errorf("Stage option %d: expected displayOrder %d, got %d", i, expected.displayOrder, actual.DisplayOrder)
			}
		}
	})

	t.Run("성공: Importance 옵션의 값과 순서 검증", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		expectedImportanceOptions := []struct {
			value        string
			label        string
			color        string
			displayOrder int
		}{
			{"urgent", "긴급", "#EF4444", 1},
			{"high", "높음", "#F97316", 2},
			{"normal", "보통", "#10B981", 3},
			{"low", "낮음", "#6B7280", 4},
		}

		importanceOptions := []fieldOptionTemplate{}
		for _, opt := range options {
			if opt.FieldType == domain.FieldTypeImportance {
				importanceOptions = append(importanceOptions, opt)
			}
		}

		for i, expected := range expectedImportanceOptions {
			if i >= len(importanceOptions) {
				t.Errorf("Missing importance option at index %d", i)
				continue
			}

			actual := importanceOptions[i]
			if actual.Value != expected.value {
				t.Errorf("Importance option %d: expected value %s, got %s", i, expected.value, actual.Value)
			}
			if actual.Label != expected.label {
				t.Errorf("Importance option %d: expected label %s, got %s", i, expected.label, actual.Label)
			}
			if actual.Color != expected.color {
				t.Errorf("Importance option %d: expected color %s, got %s", i, expected.color, actual.Color)
			}
			if actual.DisplayOrder != expected.displayOrder {
				t.Errorf("Importance option %d: expected displayOrder %d, got %d", i, expected.displayOrder, actual.DisplayOrder)
			}
		}
	})

	t.Run("성공: Role 옵션의 값과 순서 검증", func(t *testing.T) {
		// When
		options := getDefaultFieldOptions()

		// Then
		expectedRoleOptions := []struct {
			value        string
			label        string
			color        string
			displayOrder int
		}{
			{"developer", "개발자", "#8B5CF6", 1},
			{"planner", "기획자", "#EC4899", 2},
			{"designer", "디자이너", "#F59E0B", 3},
			{"qa", "QA", "#06B6D4", 4},
		}

		roleOptions := []fieldOptionTemplate{}
		for _, opt := range options {
			if opt.FieldType == domain.FieldTypeRole {
				roleOptions = append(roleOptions, opt)
			}
		}

		for i, expected := range expectedRoleOptions {
			if i >= len(roleOptions) {
				t.Errorf("Missing role option at index %d", i)
				continue
			}

			actual := roleOptions[i]
			if actual.Value != expected.value {
				t.Errorf("Role option %d: expected value %s, got %s", i, expected.value, actual.Value)
			}
			if actual.Label != expected.label {
				t.Errorf("Role option %d: expected label %s, got %s", i, expected.label, actual.Label)
			}
			if actual.Color != expected.color {
				t.Errorf("Role option %d: expected color %s, got %s", i, expected.color, actual.Color)
			}
			if actual.DisplayOrder != expected.displayOrder {
				t.Errorf("Role option %d: expected displayOrder %d, got %d", i, expected.displayOrder, actual.DisplayOrder)
			}
		}
	})
}
