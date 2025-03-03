package models

import (
	"time"
)

type Session struct {
	BaseModel
	Description      string          `gorm:"not null"`
	MaxMembers       int             `gorm:"not null"` // Maximum number of members allowed
	DateTime         *time.Time      `json:"date_time"`
	CreatedBy        string          `gorm:"not null"`
	Status           string          `gorm:"type:varchar(20);default:'open'"`
	BadmintonCourtID *string         // Foreign key to BadmintonCourt
	BadmintonCourt   *BadmintonCourt `gorm:"foreignKey:BadmintonCourtID"` // Relationship
	Attendees        []*SessionAttendee
	CreatedByName    string `gorm:"->"`
}

// ValidateSessionStatus validates the session status
func (s *Session) ValidateSessionStatus() bool {
	return ValidSessionStatus(s.Status)
}

// CanAttend checks if the session status allows attendance
func (s *Session) CanAttend() bool {
	return s.Status == SessionStatusOpen
}
