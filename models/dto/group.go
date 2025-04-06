package dto

import (
	"github.com/alanrb/badminton/backend/models"
)

type NewGroupRequest struct {
	Name     string  `json:"name"`
	ImageUrl *string `json:"image_url"`
	Remark   *string `json:"remark"`
}

type GroupResponse struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	OwnerID  string             `json:"owner_id"`
	ImageUrl *string            `json:"image_url"`
	Remark   *string            `json:"remark"`
	Members  []*UserResponse    `json:"members"`
	Sessions []*SessionResponse `json:"sessions"`
}

func ToGroupResponse(group *models.Group) *GroupResponse {
	resp := &GroupResponse{
		ID:       group.ID,
		Name:     group.Name,
		OwnerID:  group.OwnerID,
		ImageUrl: group.ImageUrl,
		Remark:   group.Remark,
	}

	for _, usr := range group.Members {
		resp.Members = append(resp.Members, &UserResponse{
			ID:        usr.ID,
			Name:      usr.Name,
			AvatarURL: usr.AvatarURL,
		})
	}

	return resp
}
