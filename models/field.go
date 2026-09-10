package models

import "time"

type FieldStatus string

const (
	FieldStatusActive      FieldStatus = "active"
	FieldStatusInactive    FieldStatus = "inactive"
	FieldStatusMaintenance FieldStatus = "maintenance"
)

func (s FieldStatus) IsValid() bool {
	return s == FieldStatusActive || s == FieldStatusInactive || s == FieldStatusMaintenance
}

type Field struct {
	ID          uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string      `gorm:"column:name;not null" json:"name"`
	Location    string      `gorm:"column:location;not null" json:"location"`
	Description string      `gorm:"column:description" json:"description"`
	Type        FieldType   `gorm:"column:type;not null" json:"type"`
	Price       float64     `gorm:"column:price;not null" json:"price"`
	Facilities  string      `gorm:"column:facilities" json:"facilities"`
	CreatedAt   time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time   `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   *time.Time  `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	Status      FieldStatus `gorm:"column:status;not null;default:'active'" json:"status"`
}

func (f *Field) IsValid() bool {
	return f.Status == FieldStatusActive
}
