package models

import "time"

// Day adalah hari operasional lapangan.
type Day string

const (
	DayMonday    Day = "MONDAY"
	DayTuesday   Day = "TUESDAY"
	DayWednesday Day = "WEDNESDAY"
	DayThursday  Day = "THURSDAY"
	DayFriday    Day = "FRIDAY"
	DaySaturday  Day = "SATURDAY"
	DaySunday    Day = "SUNDAY"
)

// Days mengembalikan seluruh hari dalam urutan Senin-Minggu.
func Days() []Day {
	return []Day{DayMonday, DayTuesday, DayWednesday, DayThursday, DayFriday, DaySaturday, DaySunday}
}

// Valid memeriksa apakah hari dikenal.
func (d Day) Valid() bool {
	for _, day := range Days() {
		if day == d {
			return true
		}
	}
	return false
}

// DayFromWeekday memetakan time.Weekday menjadi Day.
func DayFromWeekday(w time.Weekday) Day {
	switch w {
	case time.Monday:
		return DayMonday
	case time.Tuesday:
		return DayTuesday
	case time.Wednesday:
		return DayWednesday
	case time.Thursday:
		return DayThursday
	case time.Friday:
		return DayFriday
	case time.Saturday:
		return DaySaturday
	default:
		return DaySunday
	}
}

// Schedule adalah jam operasional sebuah lapangan pada hari tertentu.
// OpenTime dan CloseTime disimpan sebagai "HH:MM" agar perbandingan string
// setara dengan perbandingan waktu.
type Schedule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FieldID   uint      `gorm:"not null;index:idx_schedule_field_day,unique" json:"field_id"`
	Day       Day       `gorm:"type:varchar(10);not null;index:idx_schedule_field_day,unique" json:"day"`
	OpenTime  string    `gorm:"type:varchar(5);not null" json:"open_time"`
	CloseTime string    `gorm:"type:varchar(5);not null" json:"close_time"`
	IsClosed  bool      `gorm:"not null;default:false" json:"is_closed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Field *Field `gorm:"foreignKey:FieldID;constraint:OnDelete:CASCADE" json:"-"`
}
