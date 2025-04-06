package dto

import (
	"time"

	"github.com/alanrb/badminton/backend/models"
)

type NewSessionRequest struct {
	BadmintonCourtID string     `json:"badminton_court_id"`
	Description      string     `json:"description"`
	MaxMembers       int        `json:"max_members"`
	GroupID          string     `json:"group_id"`
	DateTime         *time.Time `json:"date_time"`
}

type UpdateSessionRequest struct {
	BadmintonCourtID string     `json:"badminton_court_id"`
	Description      string     `json:"description"`
	MaxMembers       int        `json:"max_members"`
	DateTime         *time.Time `json:"date_time"`
}

// AttendSessionRequest represents the request body for attending a session
type AttendSessionRequest struct {
	Slot int `json:"slot"`
}

type SessionResponse struct {
	CreatedAt        time.Time                  `json:"created_at"`
	ID               string                     `json:"id"`
	Description      string                     `json:"description"`
	Location         string                     `json:"location"`
	MaxMembers       int                        `json:"max_members"`
	CurrentMembers   int                        `json:"current_members"`
	DateTime         *time.Time                 `json:"date_time"`
	CreatedBy        string                     `json:"created_by"`
	CreatedByName    string                     `json:"created_by_name"`
	Status           string                     `json:"status"`
	BadmintonCourtID string                     `json:"badminton_court_id"`
	GroupName        string                     `json:"group_name"`
	Attendees        []*SessionAttendeeResponse `json:"attendees"`
}

type SessionAttendeeResponse struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Slot      int    `json:"slot"`
	Status    string `json:"status"`
	Remark    string `json:"remark"`
}

func ToSessionResponse(session *models.Session) SessionResponse {
	resp := SessionResponse{
		CreatedAt:        session.CreatedAt,
		ID:               session.ID,
		Description:      session.Description,
		MaxMembers:       session.MaxMembers,
		DateTime:         session.DateTime,
		CreatedBy:        session.CreatedBy,
		CreatedByName:    session.CreatedByName,
		Status:           session.Status,
		BadmintonCourtID: *session.BadmintonCourtID,
	}
	if session.BadmintonCourt != nil {
		resp.Location = session.BadmintonCourt.Name
	}

	if session.Group != nil {
		resp.GroupName = session.Group.Name
	}

	var currentMembers = 0
	if len(session.Attendees) > 0 {
		attendees := make([]*SessionAttendeeResponse, 0, len(session.Attendees))
		for _, attend := range session.Attendees {
			if attend.User != nil {
				attendees = append(attendees, &SessionAttendeeResponse{
					UserID:    attend.UserID,
					Name:      attend.User.Name,
					AvatarURL: attend.User.AvatarURL,
					Slot:      attend.Slot,
					Status:    string(attend.Status),
					Remark:    attend.Remark,
				})
			} else {
				attendees = append(attendees, &SessionAttendeeResponse{
					UserID: attend.UserID,
					Name:   "N/A",
					Slot:   attend.Slot,
					Status: string(attend.Status),
					Remark: attend.Remark,
				})
			}
			currentMembers += attend.Slot
		}
		resp.Attendees = attendees
	}
	resp.CurrentMembers = currentMembers

	return resp
}
