package models

import "time"

// AuditLog menyimpan jejak setiap request HTTP yang masuk ke aplikasi.
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActorID    *uint     `gorm:"index" json:"actor_id,omitempty"`
	ActorEmail string    `gorm:"type:varchar(150)" json:"actor_email"`
	ActorRole  Role      `gorm:"type:varchar(20)" json:"actor_role,omitempty"`
	Method     string    `gorm:"type:varchar(10);not null" json:"method"`
	Path       string    `gorm:"type:varchar(255);not null;index" json:"path"`
	StatusCode int       `gorm:"not null" json:"status_code"`
	DurationMS int64     `gorm:"not null" json:"duration_ms"`
	IPAddress  string    `gorm:"type:varchar(64)" json:"ip_address"`
	UserAgent  string    `gorm:"type:varchar(500)" json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}
