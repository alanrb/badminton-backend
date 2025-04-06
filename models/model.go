package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string `gorm:"primaryKey;" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

// BeforeCreate hook to generate UUID if not set
func (u *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	if len(u.ID) == 0 {
		u.ID = uuid.New().String()
	}
	return
}
