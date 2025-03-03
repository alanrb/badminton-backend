package models

type SessionAttendee struct {
	SessionID string `gorm:"primaryKey"`
	UserID    string `gorm:"primaryKey"`
	User      *User
	Slot      int // Number of slots reserved
	Status    ApprovalStatus
	Remark    string
}
