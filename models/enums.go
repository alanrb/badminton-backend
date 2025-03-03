package models

// Define the ApprovalStatus type
type ApprovalStatus string

const (
	// Session Status
	SessionStatusOpen      = "open"
	SessionStatusOngoing   = "on-going"
	SessionStatusCompleted = "completed"

	// User Role
	UserRolePlayer = "player"
	UserRoleAdmin  = "admin"

	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

// ValidSessionStatus checks if the session status is valid
func ValidSessionStatus(status string) bool {
	switch status {
	case SessionStatusOpen, SessionStatusOngoing, SessionStatusCompleted:
		return true
	default:
		return false
	}
}

// ValidUserRole checks if the user role is valid
func ValidUserRole(role string) bool {
	switch role {
	case UserRolePlayer, UserRoleAdmin:
		return true
	default:
		return false
	}
}
