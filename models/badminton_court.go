package models

import "github.com/shopspring/decimal"

type BadmintonCourt struct {
	BaseModel
	Name                 string `gorm:"not null"`
	Address              string `gorm:"not null"`
	Image                string
	GoogleMapURL         string
	EstimatePricePerHour decimal.Decimal
	Contact              string
}
