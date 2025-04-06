package models

type Group struct {
	BaseModel
	Name     string `gorm:"not null"`
	OwnerID  string `gorm:"not null"`
	ImageUrl *string
	Remark   *string
	Members  []*User    `gorm:"many2many:group_members;"`
	Sessions []*Session `gorm:"many2many:group_sessions;"`
}

type GroupMember struct {
	GroupID string `gorm:"primaryKey"`
	UserID  string `gorm:"primaryKey"`
}

type GroupSession struct {
	GroupID   string `gorm:"primaryKey"`
	SessionID string `gorm:"primaryKey"`
}
