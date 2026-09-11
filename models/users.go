package models

import "time"

// Role adalah peran pengguna dalam sistem.
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// Valid memeriksa role yang dikenal sistem.
func (r Role) Valid() bool {
	return r == RoleUser || r == RoleAdmin
}

// User adalah akun yang dapat login ke sistem.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Email     string    `gorm:"type:varchar(150);not null;uniqueIndex" json:"email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	Phone     string    `gorm:"type:varchar(20)" json:"phone"`
	Role      Role      `gorm:"type:varchar(20);not null;default:'USER';index" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Bookings []Booking `gorm:"foreignKey:UserID" json:"-"`
}

// IsAdmin menandakan user memiliki hak akses administrator.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
