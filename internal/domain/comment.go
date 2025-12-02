package domain

import "github.com/google/uuid"

// Comment represents a comment on a board
type Comment struct {
	BaseModel
	BoardID uuid.UUID `gorm:"type:uuid;not null;index:idx_comments_board_id" json:"board_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index:idx_comments_user_id" json:"user_id"`
	Content string    `gorm:"type:text;not null" json:"content"`
	Board   Board     `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE" json:"board,omitempty"`
	// ✅ 수정: Attachments는 다형성 관계이므로 FK 제거, Repository에서 별도 조회
	Attachments []Attachment `gorm:"-" json:"attachments,omitempty"`
}

// TableName specifies the table name for Comment
func (Comment) TableName() string {
	return "comments"
}
