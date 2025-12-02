package domain

import "github.com/google/uuid"

// Participant represents a user participating in a board
type Participant struct {
	BaseModel
	BoardID uuid.UUID `gorm:"type:uuid;not null;index:idx_participants_board_id;uniqueIndex:uq_participants_board_user" json:"board_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index:idx_participants_user_id;uniqueIndex:uq_participants_board_user" json:"user_id"`
	Board   Board     `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE" json:"board,omitempty"`
}

// TableName specifies the table name for Participant
func (Participant) TableName() string {
	return "participants"
}
