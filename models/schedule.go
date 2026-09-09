package models

import "time"

type Day string

const (
	Monday    Day = "monday"
	Tuesday   Day = "tuesday"
	Wednesday Day = "wednesday"
	Thursday  Day = "thursday"
	Friday    Day = "friday"
	Saturday  Day = "saturday"
	Sunday    Day = "sunday"
)

func Days() []Day {
	return []Day{
		Monday,
		Tuesday,
		Wednesday,
		Thursday,
		Friday,
		Saturday,
		Sunday,
	}
}

func (d Day) IsValid() bool {
	switch d {
	case Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday:
		return true
	default:
		return false
	}
}

func DayFromWeekday(w time.Weekday) Day {
	switch w {
	case time.Monday:
		return Monday
	case time.Tuesday:
		return Tuesday
	case time.Wednesday:
		return Wednesday
	case time.Thursday:
		return Thursday
	case time.Friday:
		return Friday
	case time.Saturday:
		return Saturday
	case time.Sunday:
		return Sunday
	default:
		return ""
	}
}

type Schedule struct {
	ID        uint      `gorm:"primarykey;autoIncrement;not null;type:bigint"`
	FieldID   uint      `gorm:"not null;type:bigint"`
	Day       Day       `gorm:"not null;type:varchar(20)"`
	OpenTime  string    `gorm:"not null;type:time"`
	CloseTime string    `gorm:"not null;type:time"`
	IsActive  bool      `gorm:"default:true;not null;type:boolean"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	Field Field `gorm:"foreignKey:FieldID;references:ID;onDelete:CASCADE;onUpdate:CASCADE"`
}
