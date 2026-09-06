package models

import "time"

// AppError is a simple application error.
type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// role dalam sistem
type role string

const (
	RoleAdmin role = "ADMIN"
	RoleUser  role = "USER"
)

// IsValid memeriksa apakah role valid
func (r role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleUser:
		return true
	default:
		return false
	}
}

// Booking represents a booking owned by a user.
type Booking struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"index" json:"user_id"`
}

// User struct mewakili entitas pengguna dalam sistem
type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Username  string `gorm:"type:varchar(100);not null" json:"username"`
	Password  string `gorm:"type:varchar(255);not null;unique" json:"password"`
	Role      role   `gorm:"type:varchar(50);not null;default:'USER'" json:"role"`
	Phone     string `gorm:"type:varchar(20)" json:"phone"`
	Email     string `gorm:"type:varchar(100);unique" json:"email"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Bookings []Booking `gorm:"foreignKey:UserID" json:"bookings"`
}

// IsValid memeriksa apakah pengguna valid
func (u *User) IsValid() bool {
	if u.Username == "" || u.Password == "" || !u.Role.IsValid() {
		return false
	}
	return true
}

// IsAdmin memeriksa apakah pengguna memiliki peran admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
