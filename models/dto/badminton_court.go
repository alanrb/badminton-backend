package dto

import (
	"github.com/alanrb/badminton/backend/models"
	"github.com/shopspring/decimal"
)

type BadmintonCourtResponse struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Address              string          `json:"address"`
	GoogleMapURL         string          `json:"google_map_url"`
	EstimatePricePerHour decimal.Decimal `json:"estimate_price_per_hour"`
	Contact              string          `json:"contact"`
}

func ToBadmintonCourtResponse(court models.BadmintonCourt) BadmintonCourtResponse {
	return BadmintonCourtResponse{
		ID:                   court.ID,
		Name:                 court.Name,
		Address:              court.Address,
		GoogleMapURL:         court.GoogleMapURL,
		EstimatePricePerHour: court.EstimatePricePerHour,
		Contact:              court.Contact,
	}
}
