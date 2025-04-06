package models

type Role struct {
	BaseModel
	Name        string        `gorm:"unique;not null" json:"name"`
	Permissions []*Permission `gorm:"many2many:role_permissions;" json:"permissions"`
}

type Permission struct {
	BaseModel
	Name string `gorm:"unique;not null" json:"name"`
}

type UserRole struct {
	UserID string `gorm:"primaryKey" json:"user_id"`
	RoleID string `gorm:"primaryKey" json:"role_id"`
}
