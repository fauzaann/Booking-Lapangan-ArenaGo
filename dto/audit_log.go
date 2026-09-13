package dto

import (
	"time"

	"Booking-Lapangan/models"
)

type AuditLogResponse struct {
	ID         uint        `json:"id"`
	ActorID    *uint       `json:"actor_id,omitempty"`
	ActorEmail string      `json:"actor_email"`
	ActorRole  models.Role `json:"actor_role,omitempty"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	StatusCode int         `json:"status_code"`
	DurationMS int64       `json:"duration_ms"`
	IPAddress  string      `json:"ip_address"`
	UserAgent  string      `json:"user_agent"`
	CreatedAt  time.Time   `json:"created_at"`
}

// NewAuditLogResponses memetakan model audit log menjadi response API.
func NewAuditLogResponses(logs []models.AuditLog) []AuditLogResponse {
	responses := make([]AuditLogResponse, 0, len(logs))
	for _, logEntry := range logs {
		responses = append(responses, AuditLogResponse{
			ID: logEntry.ID, ActorID: logEntry.ActorID, ActorEmail: logEntry.ActorEmail,
			ActorRole: logEntry.ActorRole, Method: logEntry.Method, Path: logEntry.Path,
			StatusCode: logEntry.StatusCode, DurationMS: logEntry.DurationMS,
			IPAddress: logEntry.IPAddress, UserAgent: logEntry.UserAgent, CreatedAt: logEntry.CreatedAt,
		})
	}
	return responses
}
