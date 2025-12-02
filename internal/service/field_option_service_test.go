package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func TestFieldOptionService_GetFieldOptions(t *testing.T) {
	tests := []struct {
		name        string
		fieldType   domain.FieldType
		mockRepo    func(*MockFieldOptionRepository)
		wantErr     bool
		wantErrCode string
		wantCount   int
	}{
		{
			name:      "성공: Stage 옵션 조회",
			fieldType: domain.FieldTypeStage,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return []*domain.FieldOption{
						{
							BaseModel:       domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							FieldType:       domain.FieldTypeStage,
							Value:           "pending",
							Label:           "대기",
							Color:           "#F59E0B",
							DisplayOrder:    1,
							IsSystemDefault: true,
						},
						{
							BaseModel:       domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							FieldType:       domain.FieldTypeStage,
							Value:           "in_progress",
							Label:           "진행중",
							Color:           "#3B82F6",
							DisplayOrder:    2,
							IsSystemDefault: true,
						},
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "성공: Role 옵션 조회",
			fieldType: domain.FieldTypeRole,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return []*domain.FieldOption{
						{
							BaseModel:       domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							FieldType:       domain.FieldTypeRole,
							Value:           "developer",
							Label:           "개발자",
							Color:           "#8B5CF6",
							DisplayOrder:    1,
							IsSystemDefault: true,
						},
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "성공: 빈 옵션 목록",
			fieldType: domain.FieldTypeImportance,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return []*domain.FieldOption{}, nil
				}
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "실패: 잘못된 필드 타입",
			fieldType: domain.FieldType("invalid"),
			mockRepo:  func(m *MockFieldOptionRepository) {},
			wantErr:   true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name:      "실패: DB 에러",
			fieldType: domain.FieldTypeStage,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := &MockFieldOptionRepository{}
			tt.mockRepo(mockRepo)
			service := NewFieldOptionService(mockRepo)

			// When
			got, err := service.GetFieldOptions(context.Background(), tt.fieldType)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetFieldOptions() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetFieldOptions() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetFieldOptions() unexpected error = %v", err)
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("GetFieldOptions() count = %v, want %v", len(got), tt.wantCount)
				}
			}
		})
	}
}

func TestFieldOptionService_CreateFieldOption(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.CreateFieldOptionRequest
		mockRepo    func(*MockFieldOptionRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "성공: 새로운 Stage 옵션 생성",
			req: &dto.CreateFieldOptionRequest{
				FieldType:    "stage",
				Value:        "on_hold",
				Label:        "보류",
				Color:        "#F59E0B",
				DisplayOrder: 5,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return []*domain.FieldOption{
						{
							BaseModel:    domain.BaseModel{ID: uuid.New()},
							FieldType:    domain.FieldTypeStage,
							Value:        "pending",
							Label:        "대기",
							DisplayOrder: 1,
						},
					}, nil
				}
				m.CreateFunc = func(ctx context.Context, fo *domain.FieldOption) error {
					fo.ID = uuid.New()
					fo.CreatedAt = time.Now()
					fo.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "실패: 중복된 값",
			req: &dto.CreateFieldOptionRequest{
				FieldType:    "stage",
				Value:        "pending",
				Label:        "대기",
				Color:        "#F59E0B",
				DisplayOrder: 1,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return []*domain.FieldOption{
						{
							BaseModel: domain.BaseModel{ID: uuid.New()},
							FieldType: domain.FieldTypeStage,
							Value:     "pending",
							Label:     "대기",
						},
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name: "실패: 잘못된 필드 타입",
			req: &dto.CreateFieldOptionRequest{
				FieldType:    "invalid",
				Value:        "test",
				Label:        "테스트",
				Color:        "#000000",
				DisplayOrder: 1,
			},
			mockRepo:    func(m *MockFieldOptionRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name: "실패: DB 에러 (중복 검증 중)",
			req: &dto.CreateFieldOptionRequest{
				FieldType:    "stage",
				Value:        "test",
				Label:        "테스트",
				Color:        "#000000",
				DisplayOrder: 1,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
		{
			name: "실패: DB 에러 (생성 중)",
			req: &dto.CreateFieldOptionRequest{
				FieldType:    "stage",
				Value:        "test",
				Label:        "테스트",
				Color:        "#000000",
				DisplayOrder: 1,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByFieldTypeFunc = func(ctx context.Context, ft domain.FieldType) ([]*domain.FieldOption, error) {
					return []*domain.FieldOption{}, nil
				}
				m.CreateFunc = func(ctx context.Context, fo *domain.FieldOption) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := &MockFieldOptionRepository{}
			tt.mockRepo(mockRepo)
			service := NewFieldOptionService(mockRepo)

			// When
			got, err := service.CreateFieldOption(context.Background(), tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateFieldOption() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("CreateFieldOption() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("CreateFieldOption() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("CreateFieldOption() returned nil response")
					return
				}
				if got.Value != tt.req.Value {
					t.Errorf("CreateFieldOption() Value = %v, want %v", got.Value, tt.req.Value)
				}
				if got.IsSystemDefault {
					t.Error("CreateFieldOption() user-created option should not be system default")
				}
			}
		})
	}
}

func TestFieldOptionService_UpdateFieldOption(t *testing.T) {
	optionID := uuid.New()
	newLabel := "업데이트된 라벨"
	newColor := "#FF0000"
	newOrder := 10

	tests := []struct {
		name        string
		optionID    uuid.UUID
		req         *dto.UpdateFieldOptionRequest
		mockRepo    func(*MockFieldOptionRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:     "성공: 라벨 업데이트",
			optionID: optionID,
			req: &dto.UpdateFieldOptionRequest{
				Label: &newLabel,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return &domain.FieldOption{
						BaseModel:    domain.BaseModel{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						FieldType:    domain.FieldTypeStage,
						Value:        "pending",
						Label:        "대기",
						Color:        "#F59E0B",
						DisplayOrder: 1,
					}, nil
				}
				m.UpdateFunc = func(ctx context.Context, fo *domain.FieldOption) error {
					if fo.Label != newLabel {
						return errors.New("label not updated")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:     "성공: 색상 및 순서 업데이트",
			optionID: optionID,
			req: &dto.UpdateFieldOptionRequest{
				Color:        &newColor,
				DisplayOrder: &newOrder,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return &domain.FieldOption{
						BaseModel:    domain.BaseModel{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						FieldType:    domain.FieldTypeStage,
						Value:        "pending",
						Label:        "대기",
						Color:        "#F59E0B",
						DisplayOrder: 1,
					}, nil
				}
				m.UpdateFunc = func(ctx context.Context, fo *domain.FieldOption) error {
					if fo.Color != newColor {
						return errors.New("color not updated")
					}
					if fo.DisplayOrder != newOrder {
						return errors.New("display order not updated")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:     "실패: 옵션이 존재하지 않음",
			optionID: optionID,
			req: &dto.UpdateFieldOptionRequest{
				Label: &newLabel,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name:     "실패: DB 에러 (조회 중)",
			optionID: optionID,
			req: &dto.UpdateFieldOptionRequest{
				Label: &newLabel,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
		{
			name:     "실패: DB 에러 (업데이트 중)",
			optionID: optionID,
			req: &dto.UpdateFieldOptionRequest{
				Label: &newLabel,
			},
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return &domain.FieldOption{
						BaseModel:    domain.BaseModel{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						FieldType:    domain.FieldTypeStage,
						Value:        "pending",
						Label:        "대기",
						Color:        "#F59E0B",
						DisplayOrder: 1,
					}, nil
				}
				m.UpdateFunc = func(ctx context.Context, fo *domain.FieldOption) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := &MockFieldOptionRepository{}
			tt.mockRepo(mockRepo)
			service := NewFieldOptionService(mockRepo)

			// When
			got, err := service.UpdateFieldOption(context.Background(), tt.optionID, tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateFieldOption() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("UpdateFieldOption() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("UpdateFieldOption() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("UpdateFieldOption() returned nil response")
				}
			}
		})
	}
}

func TestFieldOptionService_DeleteFieldOption(t *testing.T) {
	optionID := uuid.New()

	tests := []struct {
		name        string
		optionID    uuid.UUID
		mockRepo    func(*MockFieldOptionRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:     "성공: 사용자 생성 옵션 삭제",
			optionID: optionID,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return &domain.FieldOption{
						BaseModel:       domain.BaseModel{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						FieldType:       domain.FieldTypeStage,
						Value:           "custom",
						Label:           "커스텀",
						Color:           "#000000",
						DisplayOrder:    10,
						IsSystemDefault: false,
					}, nil
				}
				m.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:     "실패: 시스템 기본 옵션 삭제 시도",
			optionID: optionID,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return &domain.FieldOption{
						BaseModel:       domain.BaseModel{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						FieldType:       domain.FieldTypeStage,
						Value:           "pending",
						Label:           "대기",
						Color:           "#F59E0B",
						DisplayOrder:    1,
						IsSystemDefault: true,
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name:     "실패: 옵션이 존재하지 않음",
			optionID: optionID,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name:     "실패: DB 에러 (조회 중)",
			optionID: optionID,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
		{
			name:     "실패: DB 에러 (삭제 중)",
			optionID: optionID,
			mockRepo: func(m *MockFieldOptionRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.FieldOption, error) {
					return &domain.FieldOption{
						BaseModel:       domain.BaseModel{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						FieldType:       domain.FieldTypeStage,
						Value:           "custom",
						Label:           "커스텀",
						Color:           "#000000",
						DisplayOrder:    10,
						IsSystemDefault: false,
					}, nil
				}
				m.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := &MockFieldOptionRepository{}
			tt.mockRepo(mockRepo)
			service := NewFieldOptionService(mockRepo)

			// When
			err := service.DeleteFieldOption(context.Background(), tt.optionID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteFieldOption() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("DeleteFieldOption() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("DeleteFieldOption() unexpected error = %v", err)
				}
			}
		})
	}
}
