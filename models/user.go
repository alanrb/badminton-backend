package models

type AuthUser struct {
	ID     string
	Role   string
	Source string // Source of the user, e.g. "badminton", "cognito"
}

type User struct {
	BaseModel
	GoogleID  string `gorm:"unique;not null"`
	Email     string `gorm:"unique;not null"`
	Name      string `gorm:"not null"`
	Role      string `gorm:"type:varchar(20);default:'player'"`
	AvatarURL string
}

// ValidateUserRole validates the user role
func (u *User) ValidateUserRole() bool {
	return ValidUserRole(u.Role)
}
