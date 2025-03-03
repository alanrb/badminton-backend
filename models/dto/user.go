package dto

import "github.com/alanrb/badminton/backend/models"

type UserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

func ToUserResponse(user models.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Role:      user.Role,
		AvatarURL: user.AvatarURL,
	}
}
